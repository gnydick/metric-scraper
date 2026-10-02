package telemetry

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// page returns the metrics page as the handler serves it, the same path a Prometheus server reads.
func page(t *testing.T, tel *Telemetry) string {
	t.Helper()
	rec := httptest.NewRecorder()
	tel.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics page status = %d, want 200", rec.Code)
	}
	return rec.Body.String()
}

func mustNew(t *testing.T, prefix string) *Telemetry {
	t.Helper()
	tel, err := New(prefix)
	if err != nil {
		t.Fatalf("New(%q) err = %v, want nil", prefix, err)
	}
	return tel
}

// assertLines checks that each wanted line is on the page, whole.
func assertLines(t *testing.T, text string, want ...string) {
	t.Helper()
	lines := make(map[string]bool)
	for _, line := range strings.Split(text, "\n") {
		lines[line] = true
	}
	for _, line := range want {
		if !lines[line] {
			t.Errorf("the page has no line %q\npage:\n%s", line, scraperLines(text))
		}
	}
}

// scraperLines keeps the page's lines about the scraper itself, for readable failures.
func scraperLines(text string) string {
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "sink_") || strings.Contains(line, "target_") ||
			strings.Contains(line, "scans_") || strings.Contains(line, "discovery_") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// A sink's gauge is 1 while it is up and 0 while it is down. The expected lines follow from the
// name, the two labels and the stated value.
func TestSinkUpGauge(t *testing.T) {
	tel := mustNew(t, "scraper")

	tel.SinkUp("opentsdb", "tsdb:4242", true)
	assertLines(t, page(t, tel),
		"# TYPE scraper_sink_up gauge",
		`scraper_sink_up{endpoint="tsdb:4242",sink="opentsdb"} 1`)

	tel.SinkUp("opentsdb", "tsdb:4242", false)
	assertLines(t, page(t, tel), `scraper_sink_up{endpoint="tsdb:4242",sink="opentsdb"} 0`)
}

// Sink writes are counted by result: three good writes and one failed write give 3 and 1.
func TestSinkWritesCounter(t *testing.T) {
	tel := mustNew(t, "scraper")
	for i := 0; i < 3; i++ {
		tel.SinkWrite("opentsdb", nil)
	}
	tel.SinkWrite("opentsdb", errors.New("broken pipe"))

	assertLines(t, page(t, tel),
		"# TYPE scraper_sink_writes_total counter",
		`scraper_sink_writes_total{result="ok",sink="opentsdb"} 3`,
		`scraper_sink_writes_total{result="error",sink="opentsdb"} 1`)
}

// OpenTSDB's reply lines are counted by kind. Two illegal-argument replies, one unknown-metric
// reply and one reply of neither kind give 2, 1 and 1.
func TestSinkRejectionsCounter(t *testing.T) {
	tel := mustNew(t, "scraper")
	tel.SinkRejection("opentsdb", `put: illegal argument: Invalid metric name ("cpu%d")`)
	tel.SinkRejection("opentsdb", "put: illegal argument: not enough arguments")
	tel.SinkRejection("opentsdb", "put: unknown metric: No such name for 'metrics': 'cpu'")
	tel.SinkRejection("opentsdb", "something else entirely")

	assertLines(t, page(t, tel),
		"# TYPE scraper_sink_rejections_total counter",
		`scraper_sink_rejections_total{kind="illegal_argument",sink="opentsdb"} 2`,
		`scraper_sink_rejections_total{kind="unknown_metric",sink="opentsdb"} 1`,
		`scraper_sink_rejections_total{kind="other",sink="opentsdb"} 1`)
}

// A scan is counted by target kind and HTTP status, and sets its target's gauge: up only for a
// fetch that succeeded with a 2xx status.
func TestScansCounterAndTargetGauge(t *testing.T) {
	tel := mustNew(t, "scraper")
	tel.Scan("cadvisor", "node-a", 200, nil)
	tel.Scan("cadvisor", "node-a", 200, nil)
	tel.Scan("cadvisor", "node-b", 500, nil)
	tel.Scan("service", "app=x", 0, errors.New("connection refused"))

	assertLines(t, page(t, tel),
		"# TYPE scraper_scans_total counter",
		`scraper_scans_total{code="200",kind="cadvisor"} 2`,
		`scraper_scans_total{code="500",kind="cadvisor"} 1`,
		`scraper_scans_total{code="error",kind="service"} 1`,
		"# TYPE scraper_target_up gauge",
		`scraper_target_up{kind="cadvisor",target="node-a"} 1`,
		`scraper_target_up{kind="cadvisor",target="node-b"} 0`,
		`scraper_target_up{kind="service",target="app=x"} 0`)

	// A target that was down and then answers is up again.
	tel.Scan("cadvisor", "node-b", 204, nil)
	assertLines(t, page(t, tel), `scraper_target_up{kind="cadvisor",target="node-b"} 1`)
}

// Discovery rounds are counted by result: two good rounds and one failed round give 2 and 1.
func TestDiscoveryRoundsCounter(t *testing.T) {
	tel := mustNew(t, "scraper")
	tel.DiscoveryRound("cadvisor", nil)
	tel.DiscoveryRound("cadvisor", nil)
	tel.DiscoveryRound("cadvisor", errors.New("context deadline exceeded"))

	assertLines(t, page(t, tel),
		"# TYPE scraper_discovery_rounds_total counter",
		`scraper_discovery_rounds_total{kind="cadvisor",result="ok"} 2`,
		`scraper_discovery_rounds_total{kind="cadvisor",result="error"} 1`)
}

// A target that is no longer discovered leaves the page. Targets of another kind stay.
func TestKeepTargetsDropsTheOthers(t *testing.T) {
	tel := mustNew(t, "scraper")
	tel.Scan("cadvisor", "node-a", 200, nil)
	tel.Scan("cadvisor", "node-b", 200, nil)
	tel.Scan("service", "app=x", 200, nil)

	tel.KeepTargets("cadvisor", []string{"node-a"})

	text := page(t, tel)
	assertLines(t, text,
		`scraper_target_up{kind="cadvisor",target="node-a"} 1`,
		`scraper_target_up{kind="service",target="app=x"} 1`)
	if strings.Contains(text, `target="node-b"`) {
		t.Errorf("node-b is still on the page after it was dropped:\n%s", scraperLines(text))
	}

	// A dropped target that is discovered and scanned again comes back.
	tel.Scan("cadvisor", "node-b", 200, nil)
	assertLines(t, page(t, tel), `scraper_target_up{kind="cadvisor",target="node-b"} 1`)
}

// The prefix is configurable: every one of the scraper's own metric names starts with it.
func TestPrefixIsAppliedToEveryName(t *testing.T) {
	tel := mustNew(t, "acme")
	tel.SinkUp("opentsdb", "tsdb:4242", true)
	tel.SinkWrite("opentsdb", nil)
	tel.SinkRejection("opentsdb", "put: illegal argument: x")
	tel.Scan("cadvisor", "node-a", 200, nil)
	tel.DiscoveryRound("cadvisor", nil)

	text := page(t, tel)
	assertLines(t, text,
		"# TYPE acme_sink_up gauge",
		"# TYPE acme_target_up gauge",
		"# TYPE acme_sink_writes_total counter",
		"# TYPE acme_sink_rejections_total counter",
		"# TYPE acme_scans_total counter",
		"# TYPE acme_discovery_rounds_total counter")
	if strings.Contains(text, "scraper_") {
		t.Errorf("the page still holds a scraper_ name under the prefix acme:\n%s", scraperLines(text))
	}
}

// A prefix written with its trailing underscore gives the same names as one written without.
func TestPrefixMayEndInAnUnderscore(t *testing.T) {
	tel := mustNew(t, "scraper_")
	tel.SinkUp("opentsdb", "tsdb:4242", true)
	assertLines(t, page(t, tel), `scraper_sink_up{endpoint="tsdb:4242",sink="opentsdb"} 1`)
}

// A prefix that cannot start a metric name is refused.
func TestBadPrefixIsRefused(t *testing.T) {
	for _, prefix := range []string{"", "_", "9lives", "metric.scraper", "has space", "dash-ed", "colon:ed"} {
		if err := CheckPrefix(prefix); err == nil {
			t.Errorf("CheckPrefix(%q) err = nil, want an error", prefix)
		}
		if tel, err := New(prefix); err == nil || tel != nil {
			t.Errorf("New(%q) = %v, %v; want no telemetry and an error", prefix, tel, err)
		}
	}
	for _, prefix := range []string{"scraper", "scraper_", "metric_scraper", "Acme2"} {
		if err := CheckPrefix(prefix); err != nil {
			t.Errorf("CheckPrefix(%q) err = %v, want nil", prefix, err)
		}
	}
}

// A nil Telemetry records nothing and never panics, so code and tests that have no metrics page
// can pass nil.
func TestNilTelemetryRecordsNothing(t *testing.T) {
	var tel *Telemetry
	tel.SinkUp("opentsdb", "tsdb:4242", true)
	tel.SinkWrite("opentsdb", nil)
	tel.SinkRejection("opentsdb", "put: illegal argument: x")
	tel.Scan("cadvisor", "node-a", 200, nil)
	tel.DiscoveryRound("cadvisor", nil)
	tel.KeepTargets("cadvisor", nil)

	// With nothing recorded there is no page to serve: the handler answers 404, it does not panic.
	rec := httptest.NewRecorder()
	tel.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("a nil Telemetry's page answered %d, want 404", rec.Code)
	}
}
