# Alertmanager-Icinga-Bridge

The Alertmanager to Icinga bridge can receive alerts from the Prometheus Alertmanager's generic webhook receiver and creates Icinga Services for these alerts.

Other Icinga and Prometheus integrations we provide:

* https://github.com/NETWAYS/check_prometheus/
* https://github.com/NETWAYS/icinga2-exporter
* https://github.com/NETWAYS/icingaweb2-module-perfdatagraphs-prometheus
* https://github.com/NETWAYS/notify-alertmanager

## Installation

The `alertmanager-icinga-bridge` is available as an executable in the GitHub Releases and as a container image `ghcr.io/netways/alertmanager-icinga-bridge`.

* Install and start the `alertmanager-icinga-bridge` executable
* Create an Icinga host and API user for Alertmanager-Icinga-Bridge
* Create an Icinga service template for the managed services

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

## Usage

When started, Alertmanager-Icinga-Bridge listens to HTTP requests on the following paths:

* `/webhook` Endpoint to accept alerts from Alertmanager.
* `/healthz` returns HTTP 200 with `ok` as its payload as long as the webhook
  serving loop is operational.

## Configuration

```
Flags:
--help                                     Show context-sensitive help.
--id=STRING                                Instance ID ($ALERTMANAGER_ICINGA_BRIDGE_ID)
--loglevel="info"                          Loglevel (debug, info, warn, error) ($ALERTMANAGER_ICINGA_BRIDGE_LOGLEVEL)
--version                                  Print version information and quit
--icinga-url=ICINGA-URL,...                Icinga API URL (can be repeated) ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_URL)
--icinga-hostname=STRING                   Icinga host name to manage services for ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_HOSTNAME)
--disable-keep-alives                      Disable HTTP keepalives ($ALERTMANAGER_ICINGA_BRIDGE_DISABLE_KEEPALIVES)
--display-name-as-service-name             Set the Icinga service display name to the generated service name ($ALERTMANAGER_ICINGA_BRIDGE_DISPLAY_NAME_AS_SERVICE_NAME)
--icinga-insecure-tls                      Skip Icinga TLS verification ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_INSECURE_TLS)
--icinga-ca-file=STRING                    Path of a custom CA certificate to use when connecting to the Icinga API ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_CA)
--icinga-password=STRING                   Icinga API password ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_PASSWORD)
--icinga-user=STRING                       Icinga API username ($ALERTMANAGER_ICINGA_BRIDGE_ICINGA_USERNAME)
--custom-severity-levels=KEY=VALUE;...     Add or override the default mapping of severity levels to service states (severity_level=service_state) ($ALERTMANAGER_ICINGA_BRIDGE_ALERTMANAGER_CUSTOM_SEVERITY_LEVELS)
--gc-interval=15m                          Interval to check for and remove created services ($ALERTMANAGER_ICINGA_BRIDGE_GC_INTERVAL)
--heartbeat-interval=1m                    Interval for the bridge self-monitoring service heartbeat ($ALERTMANAGER_ICINGA_BRIDGE_HEARTBEAT_INTERVAL)
--heartbeat-service="heartbeat"            The name for the bridge self-monitoring service ($ALERTMANAGER_ICINGA_BRIDGE_HEARTBEAT_SERVICE)
--listen-addr="127.0.0.1:8888"             Listening address for the incoming Alertmanager requests ($ALERTMANAGER_ICINGA_BRIDGE_LISTEN_ADDR)
--bearer-token=STRING                      Bearer token for incoming requests ($ALERTMANAGER_ICINGA_BRIDGE_BEARER_TOKEN)
--tls-cert-path=STRING                     Path of a certificate file for TLS-enabled webhook endpoint (full chain) ($ALERTMANAGER_ICINGA_BRIDGE_TLS_CERT)
--tls-key-path=STRING                      Path of a private key file for TLS-enabled webhook endpoint ($ALERTMANAGER_ICINGA_BRIDGE_TLS_KEY)
--check-command="dummy"                    Specify Icinga check command during service creation ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_COMMAND)
--active-checks                            Create Icinga services as active checks ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_ACTIVE)
--plugin-output-by-states                  Enable dynamic selection of plugin output annotation based on service state ($ALERTMANAGER_ICINGA_BRIDGE_PLUGINOUTPUT_BY_STATES)
--max-check-attempts=1                     The maximum number of checks which are executed before changing to a hard state ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_MAX_CHECK_ATTEMPTS)
--templates=generic-service,...            Create Icinga services with the given template (can be repeated) ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_TEMPLATE)
--plugin-output-annotations=message,...    List of Annotation names to be used to set the plugin output for the Icinga service ($ALERTMANAGER_ICINGA_BRIDGE_PLUGINOUTPUT_ANNOTATIONS)
--checks-interval=12h                      Interval (in seconds) to be used for Icinga check_interval and retry_interval ($ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_INTERVAL)
--keep-for=168h                            How long to keep created alerts around after they have been resolved ($ALERTMANAGER_ICINGA_BRIDGE_KEEP_FOR)
--static-service-vars=KEY=VALUE;...        Custom variable to be set for craeted Icinga services (variable=value, can be repeated) ($ALERTMANAGER_ICINGA_BRIDGE_STATIC_SERVICE_VAR)
```

Most flags can be set with environment variables, refer to the help to see which flags.

The tool respects the environment variables HTTP_PROXY, HTTPS_PROXY and NO_PROXY.

## Integration to Prometheus Alertmanager

The `/webhook` endpoint accepts alerts from the Alertmanager's [generic webhook receiver](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config).

Alertmanager-Icinga-Bridge expects a the following to be part of an alert.

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

If an annotation is not found for that specific service state then Alertmanager-Icinga-Bridge will fallback on using the annotation name as configured.

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

You need to create an Icinga host which the Alertmanager-Icinga-Bridge can use to manage service's for.

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

If the key an annotation or label starts with `icinga_` it will also be added as custom variable without any prefix.

Since all labels and annotations are strings, a type information can be provided.
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

By default, the `--icinga-hostname` is used to create services and `--templates` for the service's template. This can be overridden by the following labels:

| Alert      | Icinga      |
| ---------- | ----------- |
| Label: `icinga_use_host: MyHost` | If present, use given host for the new service. The host must exist beforehand |
| Label: `icinga_use_zone: MyZone` | If present, use given zone for the new service The zone must exist beforehand |
| Label: `icinga_use_template: MyTemplate` | If present, use given template for the new service The template must exist beforehand |

Note that this requires the Alertmanager-Icinga-Bridge user to have the necessary permissions on the host.

## Heartbeat Services

Alertmanager-Icinga-Bridge supports creating "heartbeat services" in Icinga.
This can be used to map alerts like a `DeadMansSwitch`. In Prometheus a "watchdog" or "dead man's switch" is an alert that is always firing to ensure alerting pipeline is working.

To treat an alert as a "heartbeat" the alert must have a label `heartbeat` with a [Golang duration](https://pkg.go.dev/time#ParseDuration) as value (e.g. `heartbeat: "1d"`).

The Alertmanager-Icinga-Bridge will create an Icinga service check with active checks enabled and with the check interval set to the parsed duration.
We add 10% to the parsed duration to account for network latency etc., which could otherwise lead to flapping heartbeat checks.

# Thanks

This is a fork of https://github.com/vshn/signalilo
