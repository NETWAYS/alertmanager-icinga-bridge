// SPDX-License-Identifier: Apache-2.0

package icinga2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
)

const (
	icingaActionProcessCheckResultEndpoint = "/v1/actions/process-check-result/"
	icingaHostEndpoint                     = "/v1/objects/hosts/"
	icingaHostgroupEndpoint                = "/v1/objects/hostgroups/"
	icingaServiceEndpoint                  = "/v1/objects/services/"
)

var (
	ErrNotFound            = errors.New("no objects found")
	ErrNoEndpointReachable = errors.New("no Icinga endpoint reachable")
)

// Client is what we use to talk to the Icinga2 API
type Client struct {
	IcingaURL  []string
	Username   string
	Password   string
	httpClient *http.Client
	logger     *slog.Logger
	config     *config.Config
}

func NewClient(config *config.Config, logger *slog.Logger) *Client {
	var rt http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		DisableKeepAlives:   config.IcingaDisableKeepAlives,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 30 * time.Second,
		TLSClientConfig:     config.IcingaTLSConfig,
	}

	client := &http.Client{
		// We could also use a context.WithTimeout for the requests
		Timeout:   10 * time.Second,
		Transport: rt,
	}

	return &Client{
		IcingaURL:  config.IcingaURL,
		Username:   config.IcingaUser,
		Password:   config.IcingaPassword,
		httpClient: client,
		config:     config,
		logger:     logger,
	}
}

// Do is a small wrapper that tries the given request against all given Icinga API endpoints
func (c *Client) Do(req *http.Request, path string) (*http.Response, error) {
	lastErr := ErrNoEndpointReachable

	for _, base := range c.IcingaURL {
		req.URL, _ = req.URL.Parse(base + path)

		c.logger.Debug(fmt.Sprintf("Calling Icinga API at %s", req.URL), "component", "icinga")

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		req.SetBasicAuth(c.Username, c.Password)

		resp, err := c.httpClient.Do(req)

		if err != nil {
			// Got an error while trying to reach the endpoint, trying the next
			lastErr = err

			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("error %d from %s", resp.StatusCode, req.URL)

			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("all endpoints failed, last error: %w", lastErr)
}

// ProcessCheckResult handles a process-check-result for a given service
func (c *Client) ProcessCheckResult(ctx context.Context, service Service, action Action) error {
	c.logger.Debug("Processing CheckResult at Icinga API", "component", "icinga")

	action.Filter = fmt.Sprintf("host.name==\"%s\" && service.name==\"%s\"", service.HostName, service.Name)
	action.Type = "Service"

	data, errMarshal := json.Marshal(action)

	if errMarshal != nil {
		return fmt.Errorf("error encoding request: %w", errMarshal)
	}

	c.logger.Debug("Sending process-check-result", "component", "icinga", "data", string(data))

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "", bytes.NewReader(data))

	if errReq != nil {
		return errReq
	}

	res, errDo := c.Do(req, icingaActionProcessCheckResultEndpoint)

	if errDo != nil {
		return errDo
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	c.logger.Debug("Processed CheckResult at Icinga API", "component", "icinga")

	return nil
}

// GetHost returns the host that matches the given name
func (c *Client) GetHost(ctx context.Context, name string) (Host, error) {
	c.logger.Debug("Fetching host from Icinga API", "component", "icinga")

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)

	if errReq != nil {
		return Host{}, errReq
	}

	res, errDo := c.Do(req, icingaHostEndpoint+name)

	if errDo != nil {
		return Host{}, errDo
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return Host{}, ErrNotFound
	}

	bodyBytes, _ := io.ReadAll(res.Body)

	result := HostResults{}

	errDecode := json.Unmarshal(bodyBytes, &result)

	if errDecode != nil {
		return Host{}, fmt.Errorf("could not unmarshal JSON: %w", errDecode)
	}

	if len(result.Results) < 1 {
		return Host{}, ErrNotFound
	}

	host := result.Results[0].Host

	c.logger.Debug("Fetched host from Icinga API", "component", "icinga")

	return host, nil
}

// GetService returns the service that matches the given name
func (c *Client) GetService(ctx context.Context, name string) (Service, error) {
	c.logger.Debug("Fetching service from Icinga API", "component", "icinga")

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)

	if errReq != nil {
		return Service{}, errReq
	}

	res, errDo := c.Do(req, icingaServiceEndpoint+name)

	if errDo != nil {
		return Service{}, errDo
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return Service{}, ErrNotFound
	}

	bodyBytes, _ := io.ReadAll(res.Body)

	result := ServiceResults{}

	errDecode := json.Unmarshal(bodyBytes, &result)

	if errDecode != nil {
		return Service{}, fmt.Errorf("could not unmarshal JSON: %w", errDecode)
	}

	if len(result.Results) < 1 {
		return Service{}, ErrNotFound
	}

	service := result.Results[0].Service

	c.logger.Debug("Fetched service from Icinga API", "component", "icinga")

	return service, nil
}

// GetServices returns all services that match the given filter
func (c *Client) GetServices(ctx context.Context, filter QueryFilter) ([]Service, error) {
	c.logger.Debug("Fetching services from Icinga API", "component", "icinga")

	data, errMarshal := json.Marshal(filter)

	if errMarshal != nil {
		return []Service{}, fmt.Errorf("error encoding request: %w", errMarshal)
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "", bytes.NewReader(data))

	if errReq != nil {
		return []Service{}, errReq
	}

	res, errDo := c.Do(req, icingaServiceEndpoint)

	if errDo != nil {
		return []Service{}, errDo
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return []Service{}, ErrNotFound
	}

	bodyBytes, _ := io.ReadAll(res.Body)

	result := ServiceResults{}

	errDecode := json.Unmarshal(bodyBytes, &result)

	if errDecode != nil {
		return []Service{}, fmt.Errorf("could not unmarshal JSON: %w", errDecode)
	}

	services := make([]Service, 0, len(result.Results))

	for _, result := range result.Results {
		services = append(services, result.Service)
	}

	c.logger.Debug("Fetched services from Icinga API", "component", "icinga")

	return services, nil
}

// CreateService creates the given service at the Icinga API
func (c *Client) CreateService(ctx context.Context, service Service) error {
	c.logger.Debug("Creating service at Icinga API", "component", "icinga")

	serviceCreate := ServiceCreate{Templates: service.Templates, Attrs: service}
	// Strip "name" from create payload
	serviceCreate.Attrs.Name = ""

	data, errMarshal := json.Marshal(serviceCreate)

	if errMarshal != nil {
		return fmt.Errorf("error encoding request: %w", errMarshal)
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPut, "", bytes.NewReader(data))

	if errReq != nil {
		return errReq
	}

	res, errDo := c.Do(req, icingaServiceEndpoint+service.FullName())

	if errDo != nil {
		return errDo
	}

	defer res.Body.Close()

	c.logger.Debug("Created service at Icinga API", "component", "icinga")

	return nil
}

// UpdateService updates the given service at the Icinga API
func (c *Client) UpdateService(ctx context.Context, service Service) error {
	c.logger.Debug("Updating service at Icinga API", "component", "icinga")

	serviceUpdate := ServiceCreate{Attrs: service}
	// Strip "name" from update payload
	serviceUpdate.Attrs.Name = ""

	data, errMarshal := json.Marshal(serviceUpdate)

	if errMarshal != nil {
		return fmt.Errorf("error encoding request: %w", errMarshal)
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "", bytes.NewReader(data))

	if errReq != nil {
		return errReq
	}

	res, errDo := c.Do(req, icingaServiceEndpoint+service.FullName())

	if errDo != nil {
		return errDo
	}

	defer res.Body.Close()

	c.logger.Debug("Updated service at Icinga API", "component", "icinga")

	return nil
}

// DeleteService removes the given service at the Icinga API
func (c *Client) DeleteService(ctx context.Context, name string) error {
	c.logger.Debug("Deleting service at Icinga API", "component", "icinga")

	req, errReq := http.NewRequestWithContext(ctx, http.MethodDelete, "", nil)

	if errReq != nil {
		return errReq
	}

	u := url.URL{
		Path: icingaServiceEndpoint + name,
	}

	params := url.Values{"cascade": []string{"1"}}

	u.RawQuery = params.Encode()

	res, errDo := c.Do(req, u.String())

	if errDo != nil {
		return errDo
	}

	defer res.Body.Close()

	c.logger.Debug("Deleted service at Icinga API", "component", "icinga")

	return nil
}
