//nolint:revive,ireturn
package icinga2

import (
	"crypto/tls"
	"encoding/base64"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	scopeClient = iota
	scopeDialer
	scopeIcinga
	scopeTransport
)

const (
	HeaderAccept        = "Accept"
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderUserAgent     = "UserAgent"
)

const (
	MIMETypeApplicationJSON = "application/json"
)

// internal marker for builders to use in argument list
type builderOption interface {
	Scope() int
}

// clientOption is a builder option acting on [http.Client] instances
type clientOption interface {
	builderOption

	// ApplyClient applies its internal logic onto the provided instance
	ApplyClient(v *http.Client)
}

// clientOptionFunc is a [clientOption] implementation
type clientOptionFunc func(*http.Client)

// Scope implements [builderOption] and returns [scopeClient]
func (f clientOptionFunc) Scope() int {
	return scopeClient
}

// ApplyClient implements [clientOption] by calling f(v)
func (f clientOptionFunc) ApplyClient(v *http.Client) {
	f(v)
}

// WithClientTimeout sets the client timeout to the provided value
func WithClientTimeout(d time.Duration) clientOption {
	return clientOptionFunc(func(c *http.Client) {
		c.Timeout = d
	})
}

// dialerOption is a builder option acting on [net.Dialer] instances
type dialerOption interface {
	builderOption

	// ApplyDialer applies its internal logic onto the provided instance
	ApplyDialer(v *net.Dialer)
}

// dialerOptionFunc is a [dialerOption] implementation
type dialerOptionFunc func(*net.Dialer)

// Scope implements [builderOption] and returns [scopeDialer]
func (f dialerOptionFunc) Scope() int {
	return scopeDialer
}

// ApplyDialer implements [dialerOption] by calling f(v)
func (f dialerOptionFunc) ApplyDialer(v *net.Dialer) {
	f(v)
}

// WithDialerTimeout sets the dialer timeout to the provided value
func WithDialerTimeout(d time.Duration) dialerOption {
	return dialerOptionFunc(func(v *net.Dialer) {
		v.Timeout = d
	})
}

// WithDialerKeepAlive sets the KeepAlive flag on the dialer
func WithDialerKeepAlive(d time.Duration) dialerOption {
	return dialerOptionFunc(func(v *net.Dialer) {
		v.KeepAlive = d
	})
}

// icingaOption is a builder option acting on [Client] instances
type icingaOption interface {
	builderOption

	// ApplyIcinga applies its internal logic onto the provided instance
	ApplyIcinga(v *Client)
}

// icingaOptionFunc is a [icingaOption] implementation
type icingaOptionFunc func(*Client)

// Scope implements [builderOption] and returns [scopeIcinga]
func (f icingaOptionFunc) Scope() int {
	return scopeIcinga
}

// ApplyIcinga implements [icingaOption] by calling f(v)
func (f icingaOptionFunc) ApplyIcinga(v *Client) {
	f(v)
}

// WithIcingaBasicAuth adds an authorization headers using the provided
// credentials with a "Basic" scheme
func WithIcingaBasicAuth(username, password string) icingaOption {
	auth := username + ":" + password
	token := base64.StdEncoding.EncodeToString([]byte(auth))

	return icingaOptionFunc(func(v *Client) {
		v.headers[HeaderAuthorization] = "Basic " + token
	})
}

// WithIcingaUserAgent adds a user-agent header using the provided information
func WithIcingaUserAgent(application, version, metadata string) icingaOption {
	ident := application + "/" + version + " (" + metadata + ")"

	return icingaOptionFunc(func(v *Client) {
		v.headers[HeaderUserAgent] = ident
	})
}

// WithIcingaHeader add a key/value pair as header to outgoing requests
func WithIcingaHeader(key, value string) icingaOption {
	return icingaOptionFunc(func(v *Client) {
		v.headers[key] = value
	})
}

// WithIcingaAPIEndpoint adds the URL to the list of remote endpoints to connect to
func WithIcingaAPIEndpoint(url string) icingaOption {
	return icingaOptionFunc(func(v *Client) {
		v.IcingaURL = append(v.IcingaURL, url)
	})
}

// WithIcingaAPIEndpoints adds the URLs to the list of remote endpoints to connect to
func WithIcingaAPIEndpoints(urls []string) icingaOption {
	return icingaOptionFunc(func(v *Client) {
		v.IcingaURL = append(v.IcingaURL, urls...)
	})
}

// WithIcingaLogger changes the internal logger to the provided instance
func WithIcingaLogger(l *slog.Logger) icingaOption {
	return icingaOptionFunc(func(v *Client) {
		v.logger = l
	})
}

// transportOption is a builder option acting on [http.Transport] instances
type transportOption interface {
	builderOption

	// ApplyTransport applies its internal logic onto the provided instance
	ApplyTransport(v *http.Transport)
}

// transportOptionFunc is a [transportOption] implementation
type transportOptionFunc func(*http.Transport)

// Scope implements [builderOption] and returns [scopeTransport]
func (f transportOptionFunc) Scope() int {
	return scopeTransport
}

// ApplyTransport implements [transportOption] by calling f(v)
func (f transportOptionFunc) ApplyTransport(v *http.Transport) {
	f(v)
}

// WithTransportDisableKeepAlives sets the DisableKeepAlives flag on the transport instance
func WithTransportDisableKeepAlives(b bool) transportOption {
	return transportOptionFunc(func(v *http.Transport) {
		v.DisableKeepAlives = b
	})
}

// WithTransportForceAttemptHTTP2 sets the ForceAttemptHTTP2 flag on the transport instance
func WithTransportForceAttemptHTTP2(b bool) transportOption {
	return transportOptionFunc(func(v *http.Transport) {
		v.ForceAttemptHTTP2 = b
	})
}

// WithTransportTLSHandshakeTimeout sets the TLS handshake timeout on the transport instance
func WithTransportTLSHandshakeTimeout(d time.Duration) transportOption {
	return transportOptionFunc(func(v *http.Transport) {
		v.TLSHandshakeTimeout = d
	})
}

// WithTransportTLSClientConfig sets the TLS configuration on the transport instance
func WithTransportTLSClientConfig(c *tls.Config) transportOption {
	return transportOptionFunc(func(v *http.Transport) {
		v.TLSClientConfig = c
	})
}
