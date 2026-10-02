package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gnydick/metric-scraper/telemetry"
	"github.com/gnydick/metric-scraper/util/testsupport"
)

func newTestTelemetry(t *testing.T) *telemetry.Telemetry {
	t.Helper()
	tel, err := telemetry.New("scraper")
	if err != nil {
		t.Fatal(err)
	}
	return tel
}

// /healthz answers OK whenever the process is running; nothing else changes its answer
// (docs/dictated-specs/health-and-metrics.md, /healthz). It is asked here with nothing recorded,
// and again with the sink down and discovery failing.
func TestHealthzIsOKWhateverTheComponentsReport(t *testing.T) {
	tel := newTestTelemetry(t)
	router := newRouter(tel)

	ask := func(state string) {
		t.Helper()
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: /healthz status = %d, want 200", state, rec.Code)
		}
		if rec.Body.String() != "OK" {
			t.Errorf("%s: /healthz body = %q, want OK", state, rec.Body.String())
		}
	}

	ask("nothing recorded")

	tel.SinkUp("opentsdb", "tsdb:4242", false)
	tel.DiscoveryRound("cadvisor", errors.New("api server said no"))
	tel.Scan("cadvisor", "node-a", 0, errors.New("connection refused"))
	ask("sink down, discovery and scans failing")
}

// The metrics page is served at /metrics by the same router, and shows what was recorded
// (docs/dictated-specs/health-and-metrics.md, Metrics page).
func TestMetricsPageIsServedAtMetrics(t *testing.T) {
	tel := newTestTelemetry(t)
	tel.SinkUp("opentsdb", "tsdb:4242", true)

	page := testsupport.Page(t, newRouter(tel), "/metrics")

	want := `scraper_sink_up{endpoint="tsdb:4242",sink="opentsdb"} 1`
	if !testsupport.HasLine(page, want) {
		t.Errorf("GET /metrics has no line %q", want)
	}
}
