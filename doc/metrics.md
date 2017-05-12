# Using metrics

## Available metrics

Basic metrics:

* `go_*` - a set of default Go metrics provided by Prometheus library
* `process_*` - a set of default Process metrics provided by Prometheus library
* `ncagent_report_count_total` (label `agent`) - Counter. Number of total
  reports from every agent (agents separated by label).
* `ncagent_error_count_total` (label `agent`) - Counter. Number of total errors
  from every agent (agents separated by label). This counter is incremented
  when agent does not report within `reporting_interval * 2` timeframe.

HTTP probes metrics:

* `http_probe_connection_result` - Gauge. Connection result: 0 - error,
  1 - success.
* `http_probe_code` - Gauge. HTTP status code, 0 if no HTTP response.
* `http_probe_total_time_ms` - Gauge. Total duration of http request.
* `http_probe_content_transfer_time_ms` - Gauge. The duration of content
  transfer time from the first reponse byte till the end (in ms).
* `http_probe_tcp_connection_time_ms` - Gauge. TCP establishing time
  (in ms).
* `http_probe_dns_lookup_time_ms` - Gauge. DNS lookup time (in ms).
* `http_probe_connect_time_ms` - Gauge. Connection time in ms.
* `http_probe_server_processing_time_ms` - Gauge. Server processing time
  (in ms).

## Prometheus configuration example

### Scrape config

No additional configuration is needed if Prometheus has the following
configuration for PODs metrics autodiscovery:

```
# Scrape config for service endpoints.
#
# The relabeling allows the actual service scrape endpoint to be configured
# via the following annotations:
#
# * `prometheus.io/scrape`: Only scrape services that have a value of `true`
# * `prometheus.io/scheme`: If the metrics endpoint is secured then you will need
# to set this to `https` & most likely set the `tls_config` of the scrape config.
# * `prometheus.io/path`: If the metrics path is not `/metrics` override this.
# * `prometheus.io/port`: If the metrics are exposed on a different port to the
# service then set this appropriately.
- job_name: 'kubernetes-service-endpoints'

  kubernetes_sd_configs:
  - role: endpoints

  relabel_configs:
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scheme]
    action: replace
    target_label: __scheme__
    regex: (https?)
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
    action: replace
    target_label: __address__
    regex: (.+)(?::\d+)?;(\d+)
    replacement: $1:$2
  - action: labelmap
    regex: __meta_kubernetes_service_label_(.+)
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_service_name]
    action: replace
    target_label: kubernetes_name
```

The only thing which is needed in order to enable metrics gathering from
Netchecker Server is proper labeling:

```
kubectl annotate pods --selector='app==netchecker-server' \
prometheus.io/scrape=true prometheus.io/port=8081 --overwrite
```

### Alert rules configuration

* Monitoring **ncagent_error_count_total** - in this example we're firing alert
  when number for errors for the last hour becomes greater than 10:

```
ALERT NetCheckerAgentErrors
  IF absent(ncagent_error_count_total) OR
    increase(ncagent_error_count_total[1h]) > 10
  LABELS {
    service = "netchecker",
    severity = "warning"
  }
  ANNOTATIONS {
    summary = "A high number of errors in Netchecker is happening",
    description = "{{ $value }} errors have been registered within the last hour
      for Netchecker Agent {{ $labels.instance }}"
  }
```

* Monitoring **ncagent_report_count_total** - in this example we're checking that
  Netchecker Server is actually alive (not hanging or glitched). In order to do
  so we just need to check that report counter is increasing as expected.
  Report interval is 15s, so we should see at least 15 reports per 5m (ideally
  20, but due to network delays we may get less than ideal amount of reports).

```
ALERT NetCheckerReportsMissing
  IF absent(ncagent_report_count_total) OR
    increase(ncagent_report_count_total[5m]) < 15
  LABELS {
    service = "netchecker",
    severity = "warning"
  }
  ANNOTATIONS {
    summary = "The number of agent reports is lower than expected",
    description = "Netchecker Agent {{ $labels.instance }} has reported only
      {{ $value }} times for the last 5 minutes",
  }
```
