# Health and metrics

## /healthz

- Dictated by the owner (Gabe), 2026-10-01: "/healthz should be just OK if it's running, nothing conditional about it."
- `/healthz` answers OK whenever the process is running.
- Its answer depends on nothing else.
- This replaces the owner's ruling of 2026-10-01 on #7 (A hung node List stalls the scrape loop while /healthz still reports healthy), which made `/healthz` answer 503 when no node List had succeeded within two intervals.
- Tickets: #7 (closed; its `/healthz` behaviour is what this replaces), #51 (The scraper panics when it cannot connect to OpenTSDB, and exits on the first failed write), which asked whether a sink that is down should change `/healthz`. It should not.

## Metrics page

- Dictated by the owner (Gabe), 2026-10-01: "create a prometheus like metrics page with metrics and compoenent health, opentsdb.sink.one=1, opentsdb.sink.two=0 1 for up, 0 for down, and extend those metrics for return code monotonics. i may have the format of the metric names wrong, it might be snake case, find the right one"
- The scraper serves a metrics page in the Prometheus text format.
- The page holds the scraper's own metrics.
- The page holds component health.
- Component health is a gauge per component: 1 for up, 0 for down.
- The owner's example components are two OpenTSDB sinks, one up and one down.
- Those component metrics are extended with monotonic counters per return code.
- Tickets: #43 (metrics_reported on /healthz is always 0: count the metrics actually sent), #51.

## Metric name format

- The owner asked for the right format to be found.
- Source: the Prometheus documentation, read on 2026-10-01 (naming practices, writing exporters, exposition formats).
- Metric and label names use snake_case.
- A name may hold only letters, digits, underscores and colons. Colons are reserved for recording rules.
- So a dotted name such as `opentsdb.sink.one` is not a valid classic name.
- A name starts with a single-word application prefix.
- A counter's name ends in `_total`.
- A dimension goes in a label, not in the name. One name covers every component; a label says which.
- The owner's example in that form: `<prefix>_sink_up{sink="opentsdb",name="one"} 1` and `<prefix>_sink_up{sink="opentsdb",name="two"} 0`.

## Owner decisions, 2026-10-01

- Given in conversation, as answers to questions about this specification.
- Prefix: `scraper_`, and it is configurable. The owner asked about a dot (`metric.scraper_`), then chose this.
- Up/down gauges: each sink (today the one OpenTSDB sink), and each scan target.
- No up/down gauge for target discovery.
- Counters: sink writes by result; OpenTSDB rejections by kind; scans by HTTP status; discovery rounds by result.
- The page is produced with the Prometheus client library for Go. The owner approved that dependency.
- The owner then accepted the assistant's plan for the rest, and said to delete `scraper.Progress`:
  - The page is at `/metrics`, on the same port as `/healthz`.
  - `/healthz` returns the plain text `OK`. The earlier JSON report is gone.
  - The names and labels are:
    - `scraper_sink_up{sink,endpoint}`
    - `scraper_target_up{kind,target}`
    - `scraper_sink_writes_total{sink,result}`
    - `scraper_sink_rejections_total{sink,kind}`
    - `scraper_scans_total{kind,code}`
    - `scraper_discovery_rounds_total{kind,result}`
  - The prefix is set by the optional config field `metricsPrefix`. Its default is `scraper`.
  - A scan target is up when its last fetch succeeded with a 2xx status.
  - A target that is no longer discovered leaves the page.
- The owner then decided, when approving the merge:
  - While discovery itself is failing, the target gauges keep their last values.
  - The library's Go runtime and process metrics stay on the page. They do not carry the prefix.

## Glossary

- Component: a part of the scraper whose health is reported on its own, such as a sink. It is what each health gauge is about.
- Component health: whether a component is working, as 1 or 0. It is what the metrics page reports in place of a conditional `/healthz`.
- Gauge: a metric whose value can go up and down. It carries the 1 or 0 of component health.
- Monotonic counter: a metric that only ever goes up. It carries the count per return code.
- Prometheus text format: the line format a Prometheus server reads from a metrics page. It fixes how names, labels and values are written.
- Return code: the outcome a component reports for one operation. It is the label the monotonic counters are split by.
