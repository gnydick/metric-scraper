
## FILED 2026-10-01T23:38:22Z SPEC 9c3bf852-1b92-4a08-8636-7e275d70f00a

SPEC: a bad config shouldn't literally crash the service. it should be a failure from invalidated config

disposition: filed → docs/dictated-specs/config.md § Bad config at startup

## FILED 2026-10-02T00:18:13Z SPEC 9c3bf852-1b92-4a08-8636-7e275d70f00a

SPEC: /healthz should be just OK if it's running, nothing conditional about it. create a prometheus like metrics page with metrics and compoenent health, opentsdb.sink.one=1, opentsdb.sink.two=0    1 for up, 0 for down, and extend those metrics for return code monotonics. i may have the format of the metric names wrong, it might be snake case, find the right one

disposition: filed → docs/dictated-specs/health-and-metrics.md § /healthz, § Metrics page, § Metric name format
