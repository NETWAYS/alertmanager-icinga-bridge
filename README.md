# Alertmanager-Icinga-Bridge

The Alertmanager to Icinga bridge can receive alerts from the Prometheus Alertmanager's generic webhook receiver and creates Icinga Services for these alerts.

A single bridge can serve many Alertmanagers (e.g. one per Kubernetes cluster): with `--host-labels` it derives the Icinga host from alert labels such as `cluster` and `project` and creates that host on demand. See [Many Alertmanagers, hosts from alert labels](#many-alertmanagers-hosts-from-alert-labels).

Other Icinga and Prometheus integrations we provide:

* https://github.com/NETWAYS/check_prometheus/
* https://github.com/NETWAYS/icinga2-exporter
* https://github.com/NETWAYS/icingaweb2-module-perfdatagraphs-prometheus
* https://github.com/NETWAYS/notify-alertmanager

## Installation

The `alertmanager-icinga-bridge` is available as an executable in the GitHub Releases, a container image and Linux repositories.

1. Install and start the `alertmanager-icinga-bridge`
2. Create an Icinga host and API user for Alertmanager-Icinga-Bridge
3. Create an Icinga service template for the managed services

### Container

A container image is available at:

```
ghcr.io/netways/alertmanager-icinga-bridge
```

### Packages

NETWAYS provides this tool via [https://packages.netways.de](https://packages.netways.de/).

To install this module, follow the setup instructions for the **extras** repository.

**RHEL or compatible:**

`dnf install netways-alertmanager-icinga-bridge`

**Ubuntu/Debian:**

`apt install netways-alertmanager-icinga-bridge`

## Usage

When started, Alertmanager-Icinga-Bridge listens to HTTP requests on the following paths:

* `/webhook` Endpoint to accept alerts from Alertmanager.
* `/healthz` returns HTTP 200 with `ok` as its payload as long as the webhook
  serving loop is operational.

Example:

```bash
alertmanager-icinga-bridge \
 --id alertmanagerID \
 --icinga-hostname Alertmanager \
 --icinga-url "https://icinga.internal:5665" \
 --icinga-user icinga-example-api-user \
 --icinga-password icinga-example-api-password \
 --bearer-token alertmanager-example-token
```

Example for many clusters, with hosts derived from the `cluster` and `project` labels:

```bash
alertmanager-icinga-bridge \
 --id k8s \
 --icinga-hostname alertmanager-bridge \
 --icinga-url "https://localhost:5665" \
 --icinga-user alertmanager-bridge \
 --icinga-password icinga-example-api-password \
 --listen-addr 0.0.0.0:8888 \
 --tls-cert-path /etc/alertmanager-icinga-bridge/tls.crt \
 --tls-key-path /etc/alertmanager-icinga-bridge/tls.key \
 --bearer-token token-cluster-a,token-cluster-b \
 --host-labels cluster,project \
 --host-mapping-file /etc/alertmanager-icinga-bridge/host-mapping.yaml
```

## Configuration

```
Flags:
  -h, --help                       Show context-sensitive help.
      --id=STRING                  Instance ID ($ALERTMANAGER_ICINGA_BRIDGE_ID)
      --loglevel=info              Loglevel (debug, info, warn, error)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_LOGLEVEL)
      --version                    Print version information and quit
      --icinga-url=ICINGA-URL,...
                                   Icinga API URL (can be repeated)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_URL)
      --icinga-hostname=STRING     Icinga host name to manage services for
                                   ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_HOSTNAME)
      --disable-keep-alives        Disable HTTP keepalives
                                   ($ALERTMANAGER_ICINGA_BRIDGE_DISABLE_KEEPALIVES)
      --display-name-as-service-name
                                   Set the Icinga service display
                                   name to the generated service name
                                   ($ALERTMANAGER_ICINGA_BRIDGE_DISPLAY_NAME_AS_SERVICE_NAME)
      --icinga-insecure-tls        Skip Icinga TLS verification
                                   ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_INSECURE_TLS)
      --icinga-ca-file=STRING      Path of a custom CA certificate to
                                   use when connecting to the Icinga API
                                   ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_CA)
      --icinga-password=STRING     Icinga API password
                                   ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_PASSWORD)
      --icinga-user=STRING         Icinga API username
                                   ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_USERNAME)
      --custom-severity-levels=KEY=VALUE;...
                                   Add or override the default mapping
                                   of severity levels to service states
                                   (severity_level=service_state)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_ALERTMANAGER_CUSTOM_SEVERITY_LEVELS)
      --gc-interval=15m            Interval to check for and
                                   remove created services
                                   ($ALERTMANAGER_ICINGA_BRIDGE_GC_INTERVAL)
      --heartbeat-interval=1m      Interval for the bridge
                                   self-monitoring service heartbeat
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HEARTBEAT_INTERVAL)
      --heartbeat-service="heartbeat"
                                   The name for the bridge
                                   self-monitoring service
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HEARTBEAT_SERVICE)
      --listen-addr="127.0.0.1:8888"
                                   Listening address for the
                                   incoming Alertmanager requests
                                   ($ALERTMANAGER_ICINGA_BRIDGE_LISTEN_ADDR)
      --bearer-token=BEARER-TOKEN,...
                                   Bearer token for incoming requests (can
                                   be repeated, e.g. one per Alertmanager)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_BEARER_TOKEN)
      --tls-cert-path=STRING       Path of a certificate file for
                                   TLS-enabled webhook endpoint (full chain)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_TLS_CERT)
      --tls-key-path=STRING        Path of a private key file for
                                   TLS-enabled webhook endpoint
                                   ($ALERTMANAGER_ICINGA_BRIDGE_TLS_KEY)
      --check-command="dummy"      Specify Icinga check command
                                   during service creation
                                   ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_COMMAND)
      --active-checks              Create Icinga services as active checks
                                   ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_ACTIVE)
      --plugin-output-by-states    Enable dynamic selection of plugin
                                   output annotation based on service state
                                   ($ALERTMANAGER_ICINGA_BRIDGE_PLUGINOUTPUT_BY_STATES)
      --max-check-attempts=1       The maximum number of checks which are
                                   executed before changing to a hard state
                                   ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_MAX_CHECK_ATTEMPTS)
      --templates=generic-service,...
                                   Create Icinga services with the
                                   given template (can be repeated)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_TEMPLATE)
      --icinga-host-object-labels=icinga_use_host,...
                                   List of Labels depicting the Icinga
                                   Host object to associate the alert
                                   service with (uses --icinga-hostname
                                   if no label is found on alerts)
                                   ($ALERTMANAGER_BRIDGE_ICINGA_HOST_OBJECT_LABELS)
      --icinga-host-check-labels=icinga_use_host_check,...
                                   List of Labels to be used to associate
                                   the incoming alert with the Icinga
                                   Host object instead of a Service object
                                   ($ALERTMANAGER_BRIDGE_ICINGA_HOST_CHECK_LABELS)
      --icinga-host-zone-labels=icinga_use_zone,...
                                   List of Labels to be used to set the zone
                                   attribute of the Icinga service object
                                   ($ALERTMANAGER_BRIDGE_ICINGA_HOST_ZONE_LABELS)
      --icinga-host-template-labels=icinga_use_template,...
                                   List of Labels to be used to set the
                                   included template for the Icinga service
                                   object (in addition to --templates)
                                   ($ALERTMANAGER_BRIDGE_ICINGA_HOST_TEMPLATE_LABELS)
      --plugin-output-annotations=message,...
                                   List of Annotation names to be used to set
                                   the plugin output for the Icinga service
                                   ($ALERTMANAGER_ICINGA_BRIDGE_PLUGINOUTPUT_ANNOTATIONS)
      --notes-text-annotations=description,...
                                   List of Annotations to be used to set
                                   the notes text for the Icinga service
                                   ($ALERTMANAGER_BRIDGE_NOTES_TEXT_ANNOTATIONS)
      --notes-url-annotations=runbook_url,...
                                   List of Annotations to be used to set
                                   the notes URL for the Icinga service
                                   ($ALERTMANAGER_BRIDGE_NOTES_URL_ANNOTATIONS)
      --alert-fingerprint-excludes=severity,...
                                   Alert labels to exclude from calculating
                                   the fingerprint (can be repeated)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_ALERT_FINGERPRINT_EXCLUDES)
      --checks-interval=12h        Interval (in seconds) to be used for
                                   Icinga check_interval and retry_interval
                                   ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_INTERVAL)
      --keep-for=168h              How long to keep created alerts
                                   around after they have been resolved
                                   ($ALERTMANAGER_ICINGA_BRIDGE_KEEP_FOR)
      --static-service-vars=KEY=VALUE;...
                                   Custom variable to be set for created Icinga
                                   services (variable=value, can be repeated)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_STATIC_SERVICE_VAR)
      --host-labels=HOST-LABELS,...
                                   Alert labels whose values form the Icinga
                                   host name, e.g. cluster,project (hosts are
                                   created on demand; empty disables this)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_LABELS)
      --host-label-separator="-"
                                   Separator between the label
                                   values in derived host names
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_LABEL_SEPARATOR)
      --host-name-prefix=STRING    Prefix for derived host names
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_NAME_PREFIX)
      --host-mapping-file=STRING
                                   YAML/JSON file that maps label values (e.g.
                                   project IDs to names) for derived
                                   host names; reloaded on SIGHUP
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_MAPPING_FILE)
      --host-templates=generic-host,...
                                   Templates for created Icinga
                                   hosts (can be repeated)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_TEMPLATE)
      --host-check-command="dummy"
                                   Check command for created Icinga hosts
                                   (empty: leave it to the templates)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_CHECK_COMMAND)
      --host-zone=STRING           Zone for created Icinga hosts
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_ZONE)
      --static-host-vars=KEY=VALUE;...
                                   Custom variable to be set for created Icinga
                                   hosts (variable=value, can be repeated)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_STATIC_HOST_VAR)
      --host-cache-ttl=10m         How long a host is assumed to
                                   exist before it is looked up again
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_CACHE_TTL)
      --host-keep-for=168h         Remove created hosts that have no
                                   bridge services left and are older
                                   than this (0 disables host removal)
                                   ($ALERTMANAGER_ICINGA_BRIDGE_HOST_KEEP_FOR)
```

Most flags can be set with environment variables, refer to the help to see which flags.

The tool respects the environment variables HTTP_PROXY, HTTPS_PROXY and NO_PROXY.

## Integration to Prometheus Alertmanager

The `/webhook` endpoint accepts alerts from the Alertmanager's [generic webhook receiver](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config).

Alertmanager-Icinga-Bridge expects the following to be part of an alert.

Alert fields:
* `generatorURL`: Is mapped to the Icinga service `action_url`

Alert labels:
* `alertname`: Is mapped to the Icinga service `display_name` (**required**)
* `severity`: Must either be one of `warning` or `critical`, or values set via the `--custom-severity-levels` option (**required**)

Alert annotations:
* `description`: Is mapped to the Icinga service `notes` (**required**)
* `message`: Is appended to the Icinga service `plugin_output` (**required**)
* `runbook_url`: Is mapped to the Icinga service `notes_url` (**optional**)

You can also use the `--plugin-output-annotations` option to change the annotation used for the `plugin_output` as well as the `--plugin-output-by-states` option.

### Plugin Output

By default, Alertmanager-Icinga-Bridge will use the `message` annotation to set the `plugin_output` in the Icinga service.

This can be changed by using the `--plugin-output-annotations` to select either a different annotation or to provide a list of annotations where the first one with a value will be used.

Alternatively, if you enable `--plugin-output-by-states` then the Alertmanager-Icinga-Bridge will take the service state name (`ok`, `warning`, `critical`, or `unknown`) and suffix this to the annotation name when looking up the annotation to use for the plugin output (e.g. `message_ok`).

This allows you to configure multiple annotations with different values that are then used with the corresponding service state to set the plugin output.

If an annotation is not found for that specific service state then Alertmanager-Icinga-Bridge will fall back on using the annotation name as configured.

### Example Alertmanager Configuration

```yaml
global:
  resolve_timeout: 5m

route:
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h
  receiver: default
  routes:
  - match:
      alertname: DeadMansSwitch
    repeat_interval: 5m
    receiver: deadmansswitch

receivers:
- name: default
  webhook_configs:
  - send_resolved: true
    http_config:
      bearer_token: "CHANGEME"
    url: http://alertmanager.internal/webhook
- name: deadmansswitch
```

## Integration with Icinga

You need to create an Icinga host which the Alertmanager-Icinga-Bridge can use to manage services for.

Alertmanager-Icinga-Bridge expects that it has full control over this host.
Therefore, you should create a host for each Alertmanager-Icinga-Bridge instance which you're running.

Example Icinga host:

```
object Host "alertmanager.internal"  {
  display_name = "Alertmanager-Icinga-Bridge Example"
  check_command = "dummy"
  enable_passive_checks = false
  enable_perfdata = false
}
```

### Icinga service template

You need to create an Icinga service template which Alertmanager-Icinga-Bridge can use to create own services.

```
template Service "generic-service" {
}
```

### Icinga API user

We recommend that you create an API user per Icinga host.
This ensures that you create an API user per Alertmanager-Icinga-Bridge instance.

In that case, you can restrict the API user's permissions to only interact with the host belonging to the Alertmanager-Icinga-Bridge:

```
object ApiUser "alertmanager.internal"  {
  password = "CHANGEME"

  permissions = [
  {
    permission = "objects/query/*"
    filter = {{ host.name == "alertmanager.internal" }}
  },
  {
    permission = "objects/create/service"
    filter = {{ host.name == "alertmanager.internal" }}
  },
  {
    permission = "objects/modify/service"
    filter = {{ host.name == "alertmanager.internal" }}
  },
  {
    permission = "objects/delete/service"
    filter = {{ host.name == "alertmanager.internal" }}
  },
  {
    permission = "actions/process-check-result"
    filter = {{ host.name == "alertmanager.internal" }}
  }, ]
}
```

Note that you don't have to use the same name for the API user as for its associated  host.
However, you have to make sure that you compare `host.name` to the name of the service host for which the API user has permissions.

## Automatic Service Cleanup (Garbage Collection)

Service objects in Icinga will get removed on a regular basis, following these rules:

* Service object is in `OK` state
* Last transition to `OK` state was more than `--keep-for` ago
* `ID` the Alertmanager-Icinga-Bridge instances matches the service's `vars.bridge_uuid`

All state for this garbage collection is stored in Icinga service variables.

## Alertmanager-Icinga-Bridge Heartbeat

The Alertmanager-Icinga-Bridge will regularly send a passive check result to a predefined heartbeat service.
If no state update was provided, Icinga automatically marks the check as UNKNOWN.

You need to configure the following service in Icinga:

```
object Service "heartbeat" {
  check_command = "dummy"
  check_interval = 10s

  /* Set the state to CRITICAL (2) if freshness checks fail. */
  vars.dummy_state = 2

  /* Use a runtime function to retrieve the last check time and more details. */
  vars.dummy_text = {{
    var service = get_service(macro("$host.name$"), macro("$service.name$"))
    var lastCheck = DateTime(service.last_check).to_string()

    return "No check results received. Last result time: " + lastCheck
  }}

  /* This must match the name of the host object for the Alertmanager-Icinga-Bridge instance */
  host_name = "alertmanager.internal"
}
```

## Custom Variables

All alert labels and annotations will be mapped to custom variables.
Keys of labels will be prefixed with `label_` and keys of annotations with `annotation_`.

If the key of an annotation or label starts with `icinga_` it will also be added as custom variable without any prefix.

Since all labels and annotations are strings, type information can be provided.
This is done by adding the type as part of the prefix (`icinga_<type>_`).

Current supported types are `number` and `string`.

Examples:

| Alert      | Icinga      |
| ---------- | ----------- |
| Label: `foo: bar` | Custom Var: `label_foo = bar` |
| Annotation: `foo: bar` | Custom Var: `annotation_foo = bar` |
| Label: `icinga_string_foo: bar` | Custom Var: `foo = bar` |
| Annotation: `icinga_string_foo: bar` | Custom Var: `foo = bar` |
| Label: `icinga_number_foo: 123` | Custom Var: `foo = 123` |
| Annotation: `icinga_number_foo: 123` | Custom Var: `foo = 123` |

In case there is a label and an annotation with the `icinga_<type>` prefix, the value of the annotation will take precedence in the resulting set of custom variables.

## Custom Host/Zone/Template

By default, the `--icinga-hostname` is used to create services and `--templates` for the service template.
This can be overridden by the following labels (unless configured otherwise):

| Alert      | Icinga      |
| ---------- | ----------- |
| Label: `icinga_use_host: MyHost` | If present, use given host for the new service. The host must exist beforehand |
| Label: `icinga_use_host_check: "true"` | If this and `icinga_use_host` are present, set the host check result for the given host |
| Label: `icinga_use_zone: MyZone` | If present, use given zone for the new service The zone must exist beforehand |
| Label: `icinga_use_template: MyTemplate` | If present, use given template for the new service. The template must exist beforehand |

Note that, `icinga_use_host` requires the Alertmanager-Icinga-Bridge user to have the necessary permissions on the given host.
The `icinga_use_host_check` will map "firing" alerts to the Icinga Host state "down" and "resolved" to "up".

## Heartbeat Services

Alertmanager-Icinga-Bridge supports creating "heartbeat services" in Icinga.
This can be used to map alerts like a `DeadMansSwitch`. In Prometheus a "watchdog" or "dead man's switch" is an alert that is always firing to ensure alerting pipeline is working.

To treat an alert as a "heartbeat", the alert must have a label `heartbeat` with a [Golang duration](https://pkg.go.dev/time#ParseDuration) as value (e.g. `heartbeat: "1d"`).

To enable garbage collection on these alerts, they can be set to "downtime" in Icinga. A heartbeat service with an active downtime will be removed by the garbage collection.

The Alertmanager-Icinga-Bridge will create an Icinga service check with active checks enabled and with the check interval set to the parsed duration.
We add 10% to the parsed duration to account for network latency etc., which could otherwise lead to flapping heartbeat checks.

## Many Alertmanagers, hosts from alert labels

Running one bridge (and one Icinga host) per Alertmanager does not scale to dozens of clusters.
Instead, a single bridge, e.g. on the Icinga master, can receive the alerts of all Alertmanagers and
put each alert on an Icinga host derived from its labels:

```
--host-labels=cluster,project
```

| Alert labels | Icinga host |
| ------------ | ----------- |
| `cluster=xraydmz2`, `project=p-q2v7w` | `xraydmz2-p-q2v7w` |
| `cluster=xraydmz2`, `project=p-q2v7w`, with the mapping `xraydmz2/p-q2v7w: system` | `xraydmz2-system` |
| `cluster=xraydmz2` (no `project`, e.g. `Watchdog`) | `xraydmz2` |
| no `cluster` label | `--icinga-hostname` |
| `icinga_use_host=MyHost` | `MyHost` (must exist; takes precedence) |

* The values of the `--host-labels` are joined with `--host-label-separator` (default `-`) and
  prefixed with `--host-name-prefix` (default none). Characters other than letters, digits, `.`,
  `_` and `-` become `_`.
* The first label is required: alerts without it go to `--icinga-hostname`. Later labels that an
  alert does not carry are left out, unless the mapping file gives them a default.
* Make sure every Alertmanager adds the labels, e.g. as Prometheus `external_labels` or via
  `alert_relabel_configs`.

### Mapping label values

`--host-mapping-file` names a YAML (or JSON) file that translates label values, for example Rancher
project IDs to project names (see [`contrib/host-mapping.example.yaml`](contrib/host-mapping.example.yaml)):

```yaml
defaults:
  project: cluster          # optional: value for alerts without a "project" label
labels:
  project:
    xraydmz2/p-q2v7w: system   # only for cluster xraydmz2
    p-shared: shared           # for every cluster
```

A key can be scoped by the (original) values of the host labels that come before it, joined by `/`;
scoped keys take precedence over plain ones. The file is checked at startup and re-read on `SIGHUP`
(`systemctl reload`); if the new file is broken, the error is logged and the previous mapping stays.
Hosts that already exist are not renamed: alerts go to the new host name, and the old host is removed
by the garbage collection once its services are gone.

### Hosts created on demand

When an alert arrives for a host that does not exist yet, the bridge creates it with the
`--host-templates` (default `generic-host`), `--host-check-command` (default `dummy`, empty leaves it
to the templates), `--host-zone` and these custom variables:

| Variable | Value |
| -------- | ----- |
| `vars.<label>` (e.g. `vars.cluster`, `vars.project`) | label value after mapping (`system`) |
| `vars.<label>_source` (e.g. `vars.project_source`) | original value of a mapped label (`p-q2v7w`) |
| `vars.bridge_uuid` | the bridge's `--id` |
| `vars.bridge_created` | creation time (Unix seconds) |
| `--static-host-vars` | as given |

Use them in apply rules, host groups and notifications, e.g. `assign where host.vars.cluster == "xraydmz2"`.
Known hosts are cached for `--host-cache-ttl` (default 10m), so usually only the first alert for a
host costs an extra API call.

A created host is removed by the garbage collection when no service of this bridge is left on it
and it was created more than `--host-keep-for` (default 168h) ago; `--host-keep-for=0` keeps them.
Only hosts whose `vars.bridge_uuid` matches `--id` are removed, never `--icinga-hostname`.

### Authentication and transport

Every Alertmanager sends its notifications with a bearer token; `--bearer-token` can be repeated (or
given comma-separated in `ALERTMANAGER_ICINGA_BRIDGE_BEARER_TOKEN`), so each cluster can get its own
token and be revoked separately. Since the Alertmanagers run elsewhere, listen on a reachable address
(`--listen-addr 0.0.0.0:8888`) and enable TLS (`--tls-cert-path`, `--tls-key-path`).

Alertmanager configuration in each cluster:

```yaml
receivers:
- name: icinga
  webhook_configs:
  - url: https://icinga.example.com:8888/webhook
    send_resolved: true
    http_config:
      authorization:
        credentials_file: /etc/alertmanager/secrets/icinga-bridge-token
```

If one alert of a notification fails, the others are still processed and the bridge answers with
HTTP 500, so Alertmanager retries the notification.

### Icinga configuration

[`contrib/icinga2-multicluster.conf`](contrib/icinga2-multicluster.conf) has templates, host groups and an
API user whose permissions are limited to the hosts of the bridge (`host.vars.bridge_uuid`) and its own
host. The API user needs `objects/create/host` and `objects/delete/host` in addition to the service
permissions.

# Thanks

This is a fork of https://github.com/vshn/signalilo
