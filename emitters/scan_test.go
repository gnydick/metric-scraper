package emitters

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	dataCadv "github.com/gnydick/metric-scraper/data/cadvisor"
	dataSvc "github.com/gnydick/metric-scraper/data/service"
	m "github.com/gnydick/metric-scraper/metric"
	"github.com/gnydick/metric-scraper/util"
	"github.com/gnydick/metric-scraper/util/testsupport"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fakeSink is the channel end of a sink; the test reads what a scan sends.
type fakeSink struct {
	metrics chan *m.Metric
}

func (f *fakeSink) Send()                       {}
func (f *fakeSink) AddClient()                  {}
func (f *fakeSink) Wait()                       {}
func (f *fakeSink) RemoveClient()               {}
func (f *fakeSink) ClientCount() int            { return 0 }
func (f *fakeSink) GetChannel() *chan *m.Metric { return &f.metrics }

// scanAndCollect serves body, runs scan against it, and returns the "name=value" of every metric
// the scan sent, sorted, plus what it logged.
func scanAndCollect(t *testing.T, body string, scan func(url string, sink *fakeSink)) (sent []string, logged string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	sink := &fakeSink{metrics: make(chan *m.Metric)}
	logged = testsupport.CaptureStdout(t, func() {
		done := make(chan struct{})
		go func() {
			scan(srv.URL, sink)
			close(done)
		}()
		for {
			select {
			case metric := <-sink.metrics:
				sent = append(sent, fmt.Sprintf("%s=%v", metric.Metric, metric.Value))
			case <-done:
				return
			case <-time.After(30 * time.Second):
				// Generous on purpose: this only catches a hang.
				t.Error("the scan did not finish within 30s")
				return
			}
		}
	})
	sort.Strings(sent)
	return sent, logged
}

// The body both scan tests serve: two good lines, one line with a bad value, one line that is not
// in the metric format.
const scanBodyHead = "# HELP things Some things.\n# TYPE things gauge\n"

func assertSkipsLogged(t *testing.T, logged string, badValueText string, notAMetricLine string) {
	t.Helper()
	// The bad value is reported at ERROR. Off Windows the library colours the level tag.
	var errorLine string
	for _, line := range strings.Split(logged, "\n") {
		if strings.Contains(line, "ERROR") {
			errorLine = line
		}
	}
	if !strings.Contains(errorLine, badValueText) {
		t.Errorf("no ERROR line names the bad value %q: %q", badValueText, logged)
	}
	// The line that is not a metric is DEBUG only, so at the default level it is not printed.
	if strings.Contains(logged, notAMetricLine) {
		t.Errorf("the line that is not a metric was printed at the default level: %q", logged)
	}
}

// A service scan sends only the lines that parse. A bad line is skipped and logged, and the scan
// carries on (#19). Two good lines in, two metrics out.
func TestServiceScanSkipsBadLines(t *testing.T) {
	body := scanBodyHead +
		"things{kind=\"a\"} 1\n" +
		"things{kind=\"b\"} 1.2.3\n" +
		"go_goroutines 12\n" +
		"things{kind=\"c\"} 2\n"

	sent, logged := scanAndCollect(t, body, func(url string, sink *fakeSink) {
		Service{url: url, identTag: "app=x", sink: sink, serviceData: dataSvc.NewServiceData()}.Scan(context.Background())
	})

	want := []string{"things=1", "things=2"}
	if strings.Join(sent, " ") != strings.Join(want, " ") {
		t.Errorf("sent %v, want %v", sent, want)
	}
	assertSkipsLogged(t, logged, "1.2.3", "go_goroutines")
}

// Each skip is logged by its cause (#19): a line that is not a metric at DEBUG, a metric line with a
// bad tag at ERROR. The run is at LogLevel DEBUG so both lines are printed.
func TestScanLogsEachSkipByCause(t *testing.T) {
	savedLevel := util.LogLevel
	util.LogLevel = util.DEBUG
	defer func() { util.LogLevel = savedLevel }()

	body := scanBodyHead +
		"go_goroutines 12\n" +
		"things{kind=\"x=y\"} 1\n"

	sent, logged := scanAndCollect(t, body, func(url string, sink *fakeSink) {
		Service{url: url, identTag: "app=x", sink: sink, serviceData: dataSvc.NewServiceData()}.Scan(context.Background())
	})

	if len(sent) != 0 {
		t.Errorf("sent %v, want nothing: neither line is a good metric", sent)
	}
	var debugLine, errorLine string
	for _, line := range strings.Split(logged, "\n") {
		if strings.Contains(line, "go_goroutines 12") {
			debugLine = line
		}
		if strings.Contains(line, "x=y") {
			errorLine = line
		}
	}
	if !strings.Contains(debugLine, "DEBUG") || !strings.HasSuffix(debugLine, "Skipped a line that is not a metric: go_goroutines 12") {
		t.Errorf("the line that is not a metric was logged as %q, want a DEBUG line naming it: %q", debugLine, logged)
	}
	if !strings.Contains(errorLine, "ERROR") || !strings.Contains(errorLine, "Skipped a bad metric line: ") {
		t.Errorf("the bad tag was logged as %q, want an ERROR line naming it: %q", errorLine, logged)
	}
}

// The same for a cadvisor scan. The two good lines are container metrics, which the data set keeps
// under their container.
func TestCadvisorScanSkipsBadLines(t *testing.T) {
	body := scanBodyHead +
		"container_a{container_name=\"app\",pod_name=\"p\"} 1\n" +
		"container_b{container_name=\"app\",pod_name=\"p\"} 1.2.3\n" +
		"container_scrape_error 0\n" +
		"container_c{container_name=\"app\",pod_name=\"p\"} 2\n"

	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}}
	sent, logged := scanAndCollect(t, body, func(url string, sink *fakeSink) {
		Cadvisor{url: url, sink: sink, ds: dataCadv.NewDataSet(node), node: node}.Scan(context.Background())
	})

	want := []string{"container_a=1", "container_c=2"}
	if strings.Join(sent, " ") != strings.Join(want, " ") {
		t.Errorf("sent %v, want %v", sent, want)
	}
	assertSkipsLogged(t, logged, "1.2.3", "container_scrape_error")
}
