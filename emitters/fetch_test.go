package emitters

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dataCadv "github.com/gnydick/metric-scraper/data/cadvisor"
	dataSvc "github.com/gnydick/metric-scraper/data/service"
	m "github.com/gnydick/metric-scraper/metric"
	"github.com/gnydick/metric-scraper/util/testsupport"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// scanners builds each kind of emitter against url, so one test covers both.
func scanners(url string, sink *fakeSink) map[string]Emitter {
	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}}
	return map[string]Emitter{
		"cadvisor": Cadvisor{url: url, sink: sink, ds: dataCadv.NewDataSet(node), node: node},
		"service":  Service{url: url, identTag: "app=x", sink: sink, serviceData: dataSvc.NewServiceData()},
	}
}

// hangingServer answers a GET by hanging: either before any answer, or after the headers and the
// first lines of the body. reached gets a value once a request is being held.
func hangingServer(t *testing.T, afterPartialBody bool) (url string, reached <-chan struct{}) {
	t.Helper()
	held := make(chan struct{}, 4)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if afterPartialBody {
			fmt.Fprint(w, scanBodyHead)
			w.(http.Flusher).Flush()
		}
		held <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})
	return srv.URL, held
}

// A target that accepts the request and then hangs must not hold a scan past the caller's
// deadline. The scan ends, logs the failure and sends nothing (#21).
func TestScanEndsAtDeadlineWhenTheTargetHangs(t *testing.T) {
	shapes := []struct {
		name             string
		afterPartialBody bool
	}{
		{"before any answer", false},
		{"in the middle of the body", true},
	}
	for _, shape := range shapes {
		for _, kind := range []string{"cadvisor", "service"} {
			t.Run(kind+"/"+shape.name, func(t *testing.T) {
				url, reached := hangingServer(t, shape.afterPartialBody)
				sink := &fakeSink{metrics: make(chan *m.Metric)}
				emitter := scanners(url, sink)[kind]

				ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
				defer cancel()

				returned := false
				logged := testsupport.CaptureStdout(t, func() {
					done := make(chan struct{})
					go func() {
						emitter.Scan(ctx)
						close(done)
					}()
					select {
					case <-done:
						returned = true
					case metric := <-sink.metrics:
						t.Errorf("the scan sent %s from a target that never finished answering", metric.Metric)
					case <-time.After(30 * time.Second):
						// Generous on purpose: this only catches the hang, it is not the bound.
					}
				})
				if !returned {
					t.Fatal("Scan still blocked 30s after a 300ms deadline")
				}
				// The observer is alive only if the request reached the server and hung there.
				select {
				case <-reached:
				default:
					t.Fatal("Scan returned without its request reaching the server")
				}
				if !strings.Contains(logged, "ERROR") {
					t.Errorf("the timed-out scan logged no ERROR line: %q", logged)
				}
			})
		}
	}
}

// A scan does not change process-wide HTTP settings: the default transport's TLS settings are the
// same object after a scan as before (#21).
func TestScanLeavesTheDefaultTransportAlone(t *testing.T) {
	transport := http.DefaultTransport.(*http.Transport)
	before := transport.TLSClientConfig

	for _, kind := range []string{"cadvisor", "service"} {
		scanAndCollect(t, scanBodyHead+"things{kind=\"a\"} 1\n", func(url string, sink *fakeSink) {
			scanners(url, sink)[kind].Scan(context.Background())
		})
		if transport.TLSClientConfig != before {
			t.Errorf("the %s scan replaced http.DefaultTransport's TLS settings", kind)
		}
	}
}

// A service scan whose GET fails logs the error and ends. It does not panic (#21). Port 0 cannot
// be dialled, so the GET fails at once.
func TestServiceScanLogsAFailedFetchAndEnds(t *testing.T) {
	const url = "http://127.0.0.1:0/metrics"
	emitter := Service{url: url, identTag: "app=x", serviceData: dataSvc.NewServiceData()}

	logged := testsupport.CaptureStdout(t, func() { emitter.Scan(context.Background()) })

	if !strings.Contains(logged, "ERROR") || !strings.Contains(logged, url) {
		t.Errorf("the failed fetch was logged as %q, want an ERROR line naming %s", logged, url)
	}
}

// Scans still skip certificate checks, as the owner decided for #21: a target with a self-signed
// certificate is scanned. One metric line in, one metric out.
func TestScanAcceptsASelfSignedCertificate(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, scanBodyHead+"things{kind=\"a\"} 1\n")
	}))
	defer srv.Close()

	sink := &fakeSink{metrics: make(chan *m.Metric)}
	emitter := Service{url: srv.URL, identTag: "app=x", sink: sink, serviceData: dataSvc.NewServiceData()}
	done := make(chan struct{})
	go func() {
		emitter.Scan(context.Background())
		close(done)
	}()

	var sent []string
	for finished := false; !finished; {
		select {
		case metric := <-sink.metrics:
			sent = append(sent, metric.Metric)
		case <-done:
			finished = true
		case <-time.After(30 * time.Second):
			t.Fatal("the scan did not finish within 30s")
		}
	}
	if strings.Join(sent, " ") != "things" {
		t.Errorf("sent %v from a target with a self-signed certificate, want [things]", sent)
	}
}
