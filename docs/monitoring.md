# Monitoring Faythe

- [Monitoring Faythe](#monitoring-faythe)
  - [1. Health Check](#1-health-check)
  - [2. Metrics **endpoint**](#2-metrics-endpoint)
    - [2.1. Cluster](#21-cluster)
    - [2.2. API Requests](#22-api-requests)
    - [2.3. Metric backend](#23-metric-backend)
    - [2.5. Autoscaler](#25-autoscaler)
    - [2.5. Autohealer](#25-autohealer)
    - [2.6. Etcd requests](#26-etcd-requests)
    - [2.7. Golang application metrics](#27-golang-application-metrics)
  - [3. Example metrics](#3-example-metrics)

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

### 2.6. Etcd requests

| Name                               | Description                                                | Type    |
| ---------------------------------- | ---------------------------------------------------------- | ------- |
| faythe_etcd_request_failures_total | The total number of Etcd request failures (not retryable). | counter |

### 2.7. Golang application metrics

Faythe uses the `prometheus/promhttp` library's HTTP `Handler` as the handler function. Please refer [prometheus client](https://github.com/prometheus/client_golang/blob/master/prometheus/) for more details.

## 3. Example metrics

<details>
    <summary>Details</summary>

    # HELP faythe_api_in_flight_requests A gauge of requests currently being served by the wrapper handler.
    # TYPE faythe_api_in_flight_requests gauge
    faythe_api_in_flight_requests 0
    # HELP faythe_api_request_duration_seconds A histogram of latencies for requests.
    # TYPE faythe_api_request_duration_seconds histogram
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="0.05"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="0.1"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="0.25"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="0.5"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="0.75"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="1"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="2"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="5"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="20"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="60"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/",method="get",le="+Inf"} 1
    faythe_api_request_duration_seconds_sum{code="200",handler="/",method="get"} 0.000208033
    faythe_api_request_duration_seconds_count{code="200",handler="/",method="get"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="0.05"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="0.1"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="0.25"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="0.5"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="0.75"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="1"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="2"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="5"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="20"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="60"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds",method="get",le="+Inf"} 1
    faythe_api_request_duration_seconds_sum{code="200",handler="/clouds",method="get"} 0.000548039
    faythe_api_request_duration_seconds_count{code="200",handler="/clouds",method="get"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="0.05"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="0.1"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="0.25"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="0.5"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="0.75"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="1"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="2"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="5"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="20"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="60"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/clouds/openstack",method="post",le="+Inf"} 1
    faythe_api_request_duration_seconds_sum{code="200",handler="/clouds/openstack",method="post"} 0.001105333
    faythe_api_request_duration_seconds_count{code="200",handler="/clouds/openstack",method="post"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="0.05"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="0.1"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="0.25"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="0.5"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="0.75"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="2"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="5"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="20"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="60"} 1
    faythe_api_request_duration_seconds_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="+Inf"} 1
    faythe_api_request_duration_seconds_sum{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post"} 0.001696953
    faythe_api_request_duration_seconds_count{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post"} 1
    # HELP faythe_api_request_size_bytes A histogram of request sizes for requests.
    # TYPE faythe_api_request_size_bytes histogram
    faythe_api_request_size_bytes_bucket{code="200",handler="/",method="get",le="100"} 0
    faythe_api_request_size_bytes_bucket{code="200",handler="/",method="get",le="1000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/",method="get",le="10000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/",method="get",le="100000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/",method="get",le="1e+06"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/",method="get",le="1e+07"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/",method="get",le="1e+08"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/",method="get",le="+Inf"} 1
    faythe_api_request_size_bytes_sum{code="200",handler="/",method="get"} 344
    faythe_api_request_size_bytes_count{code="200",handler="/",method="get"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds",method="get",le="100"} 0
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds",method="get",le="1000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds",method="get",le="10000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds",method="get",le="100000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds",method="get",le="1e+06"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds",method="get",le="1e+07"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds",method="get",le="1e+08"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds",method="get",le="+Inf"} 1
    faythe_api_request_size_bytes_sum{code="200",handler="/clouds",method="get"} 422
    faythe_api_request_size_bytes_count{code="200",handler="/clouds",method="get"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="100"} 0
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="1000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="10000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="100000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="1e+06"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="1e+07"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="1e+08"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="+Inf"} 1
    faythe_api_request_size_bytes_sum{code="200",handler="/clouds/openstack",method="post"} 615
    faythe_api_request_size_bytes_count{code="200",handler="/clouds/openstack",method="post"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="100"} 0
    faythe_api_request_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1000"} 0
    faythe_api_request_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="10000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="100000"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1e+06"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1e+07"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1e+08"} 1
    faythe_api_request_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="+Inf"} 1
    faythe_api_request_size_bytes_sum{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post"} 1081
    faythe_api_request_size_bytes_count{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post"} 1
    # HELP faythe_api_requests_total A counter for requests to the wrapped handler.
    # TYPE faythe_api_requests_total counter
    faythe_api_requests_total{code="200",handler="/",method="get"} 1
    faythe_api_requests_total{code="200",handler="/clouds",method="get"} 1
    faythe_api_requests_total{code="200",handler="/clouds/openstack",method="post"} 1
    faythe_api_requests_total{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post"} 1
    # HELP faythe_api_response_size_bytes A histogram of response sizes for requests.
    # TYPE faythe_api_response_size_bytes histogram
    faythe_api_response_size_bytes_bucket{code="200",handler="/",method="get",le="100"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/",method="get",le="1000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/",method="get",le="10000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/",method="get",le="100000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/",method="get",le="1e+06"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/",method="get",le="1e+07"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/",method="get",le="1e+08"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/",method="get",le="+Inf"} 1
    faythe_api_response_size_bytes_sum{code="200",handler="/",method="get"} 34
    faythe_api_response_size_bytes_count{code="200",handler="/",method="get"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds",method="get",le="100"} 0
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds",method="get",le="1000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds",method="get",le="10000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds",method="get",le="100000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds",method="get",le="1e+06"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds",method="get",le="1e+07"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds",method="get",le="1e+08"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds",method="get",le="+Inf"} 1
    faythe_api_response_size_bytes_sum{code="200",handler="/clouds",method="get"} 549
    faythe_api_response_size_bytes_count{code="200",handler="/clouds",method="get"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="100"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="1000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="10000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="100000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="1e+06"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="1e+07"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="1e+08"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/clouds/openstack",method="post",le="+Inf"} 1
    faythe_api_response_size_bytes_sum{code="200",handler="/clouds/openstack",method="post"} 36
    faythe_api_response_size_bytes_count{code="200",handler="/clouds/openstack",method="post"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="100"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="10000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="100000"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1e+06"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1e+07"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="1e+08"} 1
    faythe_api_response_size_bytes_bucket{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post",le="+Inf"} 1
    faythe_api_response_size_bytes_sum{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post"} 36
    faythe_api_response_size_bytes_count{code="200",handler="/scalers/d63298fe766d54fa1de16184cd4bec35",method="post"} 1
    # HELP faythe_autohealer_workers_total The total number of healers are currently managed by this cluster member.
    # TYPE faythe_autohealer_workers_total gauge
    faythe_autohealer_workers_total{cluster="prod"} 0
    # HELP faythe_autoscaler_action_successes_total The total number of scaler action successes.
    # TYPE faythe_autoscaler_action_successes_total counter
    faythe_autoscaler_action_successes_total{cluster="prod",type="http"} 1
    # HELP faythe_autoscaler_workers_total The total number of scalers are currently managed by this cluster member.
    # TYPE faythe_autoscaler_workers_total gauge
    faythe_autoscaler_workers_total{cluster="prod"} 1
    # HELP faythe_cluster_member_info A metric with constant '1' value labeled by cluster id and member information
    # TYPE faythe_cluster_member_info gauge
    faythe_cluster_member_info{address="10.61.127.102",cluster="prod",id="4f111165872744e8b09329dcabfc51b7",name="VTN-KIENNT65"} 1
    # HELP faythe_cluster_member_join_total A counter of the number of members that have joined.
    # TYPE faythe_cluster_member_join_total counter
    faythe_cluster_member_join_total 0
    # HELP faythe_cluster_member_leave_total A counter of the number of members that have left.
    # TYPE faythe_cluster_member_leave_total counter
    faythe_cluster_member_leave_total 0
    # HELP faythe_metric_backend_query_failures_total The total number of metric backend query failures total.
    # TYPE faythe_metric_backend_query_failures_total counter
    faythe_metric_backend_query_failures_total{cluster="prod",endpoint="http://10.240.201.233:9095",type="prometheus"} 5
    # HELP faythe_etcd_request_failures_total The total number of Etcd request failures (not retryable).
    # TYPE faythe_etcd_request_failures_total counter
    faythe_etcd_request_failures_total{action="get",cluster="staging",path="/users/21232f297a57a5a743894a0e4a801fc3"} 1
    faythe_etcd_request_failures_total{action="put",cluster="staging",path="/nresolvers/87f086a3b8cab595dc69c9a4b57359ca/0f4a921b271d68158b50038b6c4c8d7b"} 1
    # HELP go_gc_duration_seconds A summary of the GC invocation durations.
    # TYPE go_gc_duration_seconds summary
    go_gc_duration_seconds{quantile="0"} 2.4959e-05
    go_gc_duration_seconds{quantile="0.25"} 2.4959e-05
    go_gc_duration_seconds{quantile="0.5"} 3.3715e-05
    go_gc_duration_seconds{quantile="0.75"} 5.0933e-05
    go_gc_duration_seconds{quantile="1"} 5.0933e-05
    go_gc_duration_seconds_sum 0.000109607
    go_gc_duration_seconds_count 3
    # HELP go_goroutines Number of goroutines that currently exist.
    # TYPE go_goroutines gauge
    go_goroutines 42
    # HELP go_info Information about the Go environment.
    # TYPE go_info gauge
    go_info{version="go1.12.7"} 1
    # HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
    # TYPE go_memstats_alloc_bytes gauge
    go_memstats_alloc_bytes 2.973872e+06
    # HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
    # TYPE go_memstats_alloc_bytes_total counter
    go_memstats_alloc_bytes_total 8.284856e+06
    # HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table.
    # TYPE go_memstats_buck_hash_sys_bytes gauge
    go_memstats_buck_hash_sys_bytes 1.4445e+06
    # HELP go_memstats_frees_total Total number of frees.
    # TYPE go_memstats_frees_total counter
    go_memstats_frees_total 37230
    # HELP go_memstats_gc_cpu_fraction The fraction of this program's available CPU time used by the GC since the program started.
    # TYPE go_memstats_gc_cpu_fraction gauge
    go_memstats_gc_cpu_fraction 1.8169142736956888e-06
    # HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata.
    # TYPE go_memstats_gc_sys_bytes gauge
    go_memstats_gc_sys_bytes 2.377728e+06
    # HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and still in use.
    # TYPE go_memstats_heap_alloc_bytes gauge
    go_memstats_heap_alloc_bytes 2.973872e+06
    # HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used.
    # TYPE go_memstats_heap_idle_bytes gauge
    go_memstats_heap_idle_bytes 6.1628416e+07
    # HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use.
    # TYPE go_memstats_heap_inuse_bytes gauge
    go_memstats_heap_inuse_bytes 4.530176e+06
    # HELP go_memstats_heap_objects Number of allocated objects.
    # TYPE go_memstats_heap_objects gauge
    go_memstats_heap_objects 14137
    # HELP go_memstats_heap_released_bytes Number of heap bytes released to OS.
    # TYPE go_memstats_heap_released_bytes gauge
    go_memstats_heap_released_bytes 0
    # HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system.
    # TYPE go_memstats_heap_sys_bytes gauge
    go_memstats_heap_sys_bytes 6.6158592e+07
    # HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
    # TYPE go_memstats_last_gc_time_seconds gauge
    go_memstats_last_gc_time_seconds 1.5762918055096064e+09
    # HELP go_memstats_lookups_total Total number of pointer lookups.
    # TYPE go_memstats_lookups_total counter
    go_memstats_lookups_total 0
    # HELP go_memstats_mallocs_total Total number of mallocs.
    # TYPE go_memstats_mallocs_total counter
    go_memstats_mallocs_total 51367
    # HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures.
    # TYPE go_memstats_mcache_inuse_bytes gauge
    go_memstats_mcache_inuse_bytes 6944
    # HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system.
    # TYPE go_memstats_mcache_sys_bytes gauge
    go_memstats_mcache_sys_bytes 16384
    # HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures.
    # TYPE go_memstats_mspan_inuse_bytes gauge
    go_memstats_mspan_inuse_bytes 50256
    # HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system.
    # TYPE go_memstats_mspan_sys_bytes gauge
    go_memstats_mspan_sys_bytes 65536
    # HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place.
    # TYPE go_memstats_next_gc_bytes gauge
    go_memstats_next_gc_bytes 4.194304e+06
    # HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations.
    # TYPE go_memstats_other_sys_bytes gauge
    go_memstats_other_sys_bytes 1.273444e+06
    # HELP go_memstats_stack_inuse_bytes Number of bytes in use by the stack allocator.
    # TYPE go_memstats_stack_inuse_bytes gauge
    go_memstats_stack_inuse_bytes 950272
    # HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator.
    # TYPE go_memstats_stack_sys_bytes gauge
    go_memstats_stack_sys_bytes 950272
    # HELP go_memstats_sys_bytes Number of bytes obtained from system.
    # TYPE go_memstats_sys_bytes gauge
    go_memstats_sys_bytes 7.2286456e+07
    # HELP go_threads Number of OS threads created.
    # TYPE go_threads gauge
    go_threads 15
    # HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
    # TYPE process_cpu_seconds_total counter
    process_cpu_seconds_total 932.81
    # HELP process_max_fds Maximum number of open file descriptors.
    # TYPE process_max_fds gauge
    process_max_fds 1024
    # HELP process_open_fds Number of open file descriptors.
    # TYPE process_open_fds gauge
    process_open_fds 14
    # HELP process_resident_memory_bytes Resident memory size in bytes.
    # TYPE process_resident_memory_bytes gauge
    process_resident_memory_bytes 1.9349504e+07
    # HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
    # TYPE process_start_time_seconds gauge
    process_start_time_seconds 1.57629143968e+09
    # HELP process_virtual_memory_bytes Virtual memory size in bytes.
    # TYPE process_virtual_memory_bytes gauge
    process_virtual_memory_bytes 1.111883776e+09
    # HELP process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
    # TYPE process_virtual_memory_max_bytes gauge
    process_virtual_memory_max_bytes -1
    # HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
    # TYPE promhttp_metric_handler_requests_in_flight gauge
    promhttp_metric_handler_requests_in_flight 1
    # HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
    # TYPE promhttp_metric_handler_requests_total counter
    promhttp_metric_handler_requests_total{code="200"} 4
    promhttp_metric_handler_requests_total{code="500"} 0
    promhttp_metric_handler_requests_total{code="503"} 0

</details>
