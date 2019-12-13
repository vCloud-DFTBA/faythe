# Monitoring Faythe

## 1. Health Check

Faythe also provides the simple `/healthz` endpoint with the member uptime.

## 2. Metrics **endpoint**

Faythe uses [Prometheus](https://prometheus.io/) for metrics reporting. The metrics can be used for real-monitoring and debugging. Faythe does not persist its metrics; if a member restarts, the metrics will be reset.

The simplest way to see the available metrics is to cURL the metrics endpoint `/metrics` on its client port. The format is described [here](https://prometheus.io/docs/instrumenting/exposition_formats/).

Follow the [Prometheus getting started doc](https://prometheus.io/docs/prometheus/latest/getting_started/) to spin up a Prometheus server to collect Faythe metrics.

The naming of metrics follows the suggested [Prometheus best practices](https://prometheus.io/docs/practices/naming/). A metric name has an `faythe` prefix as its namespace and a subsystem prefix (for example `cluster`).

### 2.1. Cluster

| Name                              | Description                                          | Type    |
| --------------------------------- | ---------------------------------------------------- | ------- |
| faythe_cluster_member_join_total  | A counter of the number of members that have joined. | counter |
| faythe_cluster_member_leave_total | A counter of the number of members that have left.   | counter |

### 2.2. API Requests

| Name                                | Description                                                        | Type      |
| ----------------------------------- | ------------------------------------------------------------------ | --------- |
| faythe_api_in_flight_requests       | A gauge of requests currently being served by the wrapper handler. | gauge     |
| faythe_api_requests_total           | A counter for requests to the wrapped handler.                     | counter   |
| faythe_api_request_duration_seconds | A histogram of latencies for requests.                             | histogram |
| faythe_api_request_size_bytes       | A histogram of request sizes for requests.                         | histogram |
| faythe_api_response_size_bytes      | A histogram of response sizes for requests.                        | histogram |

### 2.3. Metric backend

| Name                                       | Description                                              | Type    |
| ------------------------------------------ | -------------------------------------------------------- | ------- |
| faythe_metric_backend_query_failures_total | The total number of metric backend query failures total. | counter |

### 2.4. Automation Engine backend

| Name                           | Description                                                                       | Type    |
| ------------------------------ | --------------------------------------------------------------------------------- | ------- |
| faythe_at_query_failures_total | The total number of automation system (Stackstorm for ex) request failures total. | counter |

### 2.5. Autoscaler

| Name                                     | Description                                                               | Type    |
| ---------------------------------------- | ------------------------------------------------------------------------- | ------- |
| faythe_autoscaler_workers_total          | The total number of scalers are currently managed by this cluster member. | gauge   |
| faythe_autoscaler_action_failures_total  | The total number of scaler action failures.                               | counter |
| faythe_autoscaler_action_successes_total | The total number of scaler action successes.                              | counter |

### 2.5. Autohealer

| Name                                     | Description                                                               | Type    |
| ---------------------------------------- | ------------------------------------------------------------------------- | ------- |
| faythe_autohealer_workers_total          | The total number of healers are currently managed by this cluster member. | gauge   |
| faythe_autohealer_action_successes_total | The total number of healers action successes.                             | counter |
| faythe_autohealer_action_failures_total  | The total number of healers action failures.                              | counter |

### 2.6. Golang application metrics

Faythe uses the `prometheus/promhttp` library's HTTP `Handler` as the handler function. Please refer [prometheus client](https://github.com/prometheus/client_golang/blob/master/prometheus/) for more details.
