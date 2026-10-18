# Prometheus Alertmanager / Icinga Bridge

Test stack of components required to bring Prometheus Alertmanager alerts into
Icinga2. This particular setup uses Prometheus as alert source

## Components

The alerting landscape consists of user facing- and support services.
The former are accessible via web browser locally.

### Alertmanager Prometheus Bridge

Binding glue between Prometheus Alertmanager and Icinga2.

### Icinga2

Monitoring system; actively schedules host/service checks and accepts results
via HTTP API from Alertmanager Prometheus Bridge.

### IcingaDB

Icinga2 worker queue to process monitoring events.

### Postgres

RDBMS used by IcingaDB and Icingaweb2 as permanent storage.

### Valkey

Key-value store used by IcingaDB as intermediate storage.

### Prometheus

Metrics collector and storage. The web interface is
accessible at [http://localhost:9090](http://localhost:9090)

### Prometheus Alertmanager

Prometheus Alertmanager is receiving alerts either from Prometheus or locally
via helper script (`./fire.sh`). The web interface is accessible
at [http://localhost:9093](http://localhost:9093)

### Icingaweb2

Web interface for Icinga2 accessible at
[http://localhost:8080](http://localhost:8080), with credentials
for the administrator account being `icingaadmin`/`icingaadmin`.

## Alerts

The alerts are for demonstration purposes only and do not necessarily
represent real monitoring scenarios.

### PrometheusDiscoveryHighRequestRate

Visit the discovery endpoints in the Prometheus web interface to
trigger this synthetic alert; refresh the page to resolve the alert:

* http://localhost:9090/alertmanager-discovery
    * http://localhost:8080/icingadb/service?name=PrometheusDiscoveryHighRequestRate_a274e53dc7c35811&host.name=prometheus.localhost
* http://localhost:9090/service-discovery
    * http://localhost:8080/icingadb/service?name=PrometheusDiscoveryHighRequestRate_a5b0cfc1ec22ef1c&host.name=prometheus.localhost

### Watchdog

An alert to report on the state of the alerting pipeline from back to front.
It constantly fires in Prometheus to act as heartbeat. Stopping the Prometheus
container or otherwise interrupting the alert dispatching will eventually
result in the service state to reflect this:

* http://localhost:8080/dashboard#!/icingadb/service?name=Watchdog_664753abe3262835&host.name=bridge.localhost

### AlertmanagerUp

Example alert to showcase mapping alerts to host objects in Icinga2.
Visit the Prometheus Alertmanager status page to trigger this
synthetic alert; refresh the page to resolve the alert:

* http://localhost:9093/#/status
    * http://localhost:8080/icingadb/host?name=alertmanager.localhost

### External Connectivity (heartbeat)

Not an alert in the Prometheus configuration but a heartbeat signal created
and managed exclusively by the Alertmanager Prometheus Bridge:

* http://localhost:8080/icingadb/service?name=heartbeat&host.name=bridge.localhost
