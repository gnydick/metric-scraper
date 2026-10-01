package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	s "github.com/gnydick/metric-scraper/scraper"
)

// /healthz answers 503 when no target discovery succeeded within two scrape intervals, and 200
// otherwise (#7). The handler reads the same Progress the scrape loop writes.
func TestHealthzReflectsScrapeProgress(t *testing.T) {
	interval := 10 * time.Second

	cases := []struct {
		name string
		// sinceSuccess is how long ago the last success was when /healthz is asked.
		sinceSuccess time.Duration
		wantStatus   int
	}{
		{"success one interval ago", interval, http.StatusOK},
		{"no success for an hour", time.Hour, http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			progress := s.NewProgress(time.Now().Add(-tc.sinceSuccess), interval)
			handler := healthzHandler(func() bool { return progress.Healthy(time.Now()) })

			rec := httptest.NewRecorder()
			handler(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			// The body stays the JSON report in both cases.
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body %q is not the JSON health report: %v", rec.Body.String(), err)
			}
			if _, ok := body["uptime"]; !ok {
				t.Errorf("body %v has no uptime", body)
			}
		})
	}
}
