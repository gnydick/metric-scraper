// Package telemetry holds the scraper's own metrics and serves them as a Prometheus metrics page
// (docs/dictated-specs/health-and-metrics.md).
package telemetry

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// DefaultPrefix starts every metric name when the config names no other.
const DefaultPrefix = "scraper"

var prefixPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// normalPrefix drops the underscore a prefix may be written with: "scraper_" and "scraper" both
// give names such as scraper_sink_up.
func normalPrefix(prefix string) string {
	return strings.TrimRight(prefix, "_")
}

// CheckPrefix says whether prefix can start a metric name: a letter, then letters, digits and
// underscores. It is the one definition of a valid prefix; the config and New both use it.
func CheckPrefix(prefix string) error {
	if !prefixPattern.MatchString(normalPrefix(prefix)) {
		return fmt.Errorf("metrics prefix %q must start with a letter and hold only letters, digits and underscores", prefix)
	}
	return nil
}

// Telemetry records what the scraper does and serves it as a metrics page. New is the one way to
// get a working one. A nil *Telemetry is valid and records nothing, so code that has no metrics
// page to feed can pass nil.
type Telemetry struct {
	registry *prometheus.Registry

	sinkUp          *prometheus.GaugeVec
	targetUp        *prometheus.GaugeVec
	sinkWrites      *prometheus.CounterVec
	sinkRejections  *prometheus.CounterVec
	scans           *prometheus.CounterVec
	discoveryRounds *prometheus.CounterVec

	// targets holds, per kind, the targets that have a gauge on the page, so KeepTargets knows
	// which to drop.
	mu      sync.Mutex
	targets map[string]map[string]struct{}
}

// New builds the scraper's metrics under prefix.
func New(prefix string) (*Telemetry, error) {
	if err := CheckPrefix(prefix); err != nil {
		return nil, err
	}
	namespace := normalPrefix(prefix)

	t := &Telemetry{
		registry: prometheus.NewRegistry(),
		sinkUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "sink_up",
			Help: "Whether the sink is up: 1 for up, 0 for down.",
		}, []string{"sink", "endpoint"}),
		targetUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "target_up",
			Help: "Whether the last scan of the target succeeded: 1 for up, 0 for down.",
		}, []string{"kind", "target"}),
		sinkWrites: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "sink_writes_total",
			Help: "Metrics written to the sink, by result.",
		}, []string{"sink", "result"}),
		sinkRejections: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "sink_rejections_total",
			Help: "Metrics the sink rejected, by kind of reply.",
		}, []string{"sink", "kind"}),
		scans: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "scans_total",
			Help: "Scans of targets, by target kind and HTTP status code.",
		}, []string{"kind", "code"}),
		discoveryRounds: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "discovery_rounds_total",
			Help: "Target discovery rounds, by result.",
		}, []string{"kind", "result"}),
		targets: make(map[string]map[string]struct{}),
	}
	t.registry.MustRegister(
		t.sinkUp, t.targetUp, t.sinkWrites, t.sinkRejections, t.scans, t.discoveryRounds,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return t, nil
}

// Handler serves the metrics page.
func (t *Telemetry) Handler() http.Handler {
	return promhttp.HandlerFor(t.registry, promhttp.HandlerOpts{Registry: t.registry})
}

func result(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

// SinkUp records whether the sink at endpoint is up.
func (t *Telemetry) SinkUp(sink string, endpoint string, up bool) {
	if t == nil {
		return
	}
	t.sinkUp.WithLabelValues(sink, endpoint).Set(upValue(up))
}

// SinkWrite counts one metric written to the sink, or one that failed to be written.
func (t *Telemetry) SinkWrite(sink string, err error) {
	if t == nil {
		return
	}
	t.sinkWrites.WithLabelValues(sink, result(err)).Inc()
}

// SinkRejection counts one reply line in which the sink rejected a metric.
func (t *Telemetry) SinkRejection(sink string, reply string) {
	if t == nil {
		return
	}
	kind := "other"
	switch {
	case strings.Contains(reply, "illegal argument"):
		kind = "illegal_argument"
	case strings.Contains(reply, "unknown metric"):
		kind = "unknown_metric"
	}
	t.sinkRejections.WithLabelValues(sink, kind).Inc()
}

// Scan counts one scan of target and records whether the target is up. A scan whose fetch failed
// is counted under the code "error". The target is up only for a 2xx status.
func (t *Telemetry) Scan(kind string, target string, status int, err error) {
	if t == nil {
		return
	}
	code := "error"
	if err == nil {
		code = strconv.Itoa(status)
	}
	t.scans.WithLabelValues(kind, code).Inc()

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.targets[kind] == nil {
		t.targets[kind] = make(map[string]struct{})
	}
	t.targets[kind][target] = struct{}{}
	t.targetUp.WithLabelValues(kind, target).Set(upValue(err == nil && status >= 200 && status < 300))
}

// DiscoveryRound counts one round of target discovery.
func (t *Telemetry) DiscoveryRound(kind string, err error) {
	if t == nil {
		return
	}
	t.discoveryRounds.WithLabelValues(kind, result(err)).Inc()
}

// KeepTargets drops the gauge of every target of this kind that is not in targets, so a target
// that is no longer discovered leaves the page.
func (t *Telemetry) KeepTargets(kind string, targets []string) {
	if t == nil {
		return
	}
	keep := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		keep[target] = struct{}{}
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	for target := range t.targets[kind] {
		if _, kept := keep[target]; !kept {
			t.targetUp.DeleteLabelValues(kind, target)
			delete(t.targets[kind], target)
		}
	}
}

func upValue(up bool) float64 {
	if up {
		return 1
	}
	return 0
}
