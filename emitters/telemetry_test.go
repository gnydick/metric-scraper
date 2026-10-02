package emitters

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dataCadv "github.com/gnydick/metric-scraper/data/cadvisor"
	dataSvc "github.com/gnydick/metric-scraper/data/service"
	m "github.com/gnydick/metric-scraper/metric"
	"github.com/gnydick/metric-scraper/telemetry"
	"github.com/gnydick/metric-scraper/util/testsupport"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// scanOnce runs one scan of each kind of emitter against url, with tel recording, and drains what
// the scans send.
func scanOnce(t *testing.T, url string, tel *telemetry.Telemetry) {
	t.Helper()
	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}}
	sink := &fakeSink{metrics: make(chan *m.Metric)}
	emitters := []Emitter{
		Cadvisor{url: url, sink: sink, ds: dataCadv.NewDataSet(node), node: node, telemetry: tel},
		Service{url: url, identTag: "app=x", sink: sink, serviceData: dataSvc.NewServiceData(), telemetry: tel},
	}
	testsupport.CaptureStdout(t, func() {
		for _, emitter := range emitters {
			done := make(chan struct{})
			go func(emitter Emitter) {
				emitter.Scan(context.Background())
				close(done)
			}(emitter)
			for finished := false; !finished; {
				select {
				case <-sink.metrics:
				case <-done:
					finished = true
				case <-time.After(30 * time.Second):
					t.Error("the scan did not finish within 30s")
					finished = true
				}
			}
		}
	})
}

// Each scan is counted by target kind and HTTP status, and sets its target's up gauge (#53). The
// expected lines follow from the status the stand-in target answers with.
func TestScanRecordsItsStatusAndTargetHealth(t *testing.T) {
	cases := []struct {
		name   string
		status int
		code   string
		up     string
	}{
		{"a target that answers 200 is up", http.StatusOK, "200", "1"},
		{"a target that answers 500 is down", http.StatusInternalServerError, "500", "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, scanBodyHead+"things{container_name=\"app\",pod_name=\"p\"} 1\n")
			}))
			defer srv.Close()
			tel, err := telemetry.New("scraper")
			if err != nil {
				t.Fatal(err)
			}

			scanOnce(t, srv.URL, tel)

			page := testsupport.Page(t, tel.Handler(), "/metrics")
			for _, want := range []string{
				`scraper_scans_total{code="` + tc.code + `",kind="cadvisor"} 1`,
				`scraper_scans_total{code="` + tc.code + `",kind="service"} 1`,
				`scraper_target_up{kind="cadvisor",target="node-a"} ` + tc.up,
				`scraper_target_up{kind="service",target="app=x"} ` + tc.up,
			} {
				if !testsupport.HasLine(page, want) {
					t.Errorf("the metrics page has no line %q", want)
				}
			}
		})
	}
}

// A scan whose fetch fails is counted under the code "error" and marks its target down (#53). Port
// 0 cannot be dialled, so the fetch fails at once.
func TestScanRecordsAFailedFetch(t *testing.T) {
	tel, err := telemetry.New("scraper")
	if err != nil {
		t.Fatal(err)
	}

	scanOnce(t, "http://127.0.0.1:0/metrics", tel)

	page := testsupport.Page(t, tel.Handler(), "/metrics")
	for _, want := range []string{
		`scraper_scans_total{code="error",kind="cadvisor"} 1`,
		`scraper_scans_total{code="error",kind="service"} 1`,
		`scraper_target_up{kind="cadvisor",target="node-a"} 0`,
		`scraper_target_up{kind="service",target="app=x"} 0`,
	} {
		if !testsupport.HasLine(page, want) {
			t.Errorf("the metrics page has no line %q", want)
		}
	}
}
