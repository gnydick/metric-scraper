package emitters

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"

	m "github.com/gnydick/metric-scraper/metric"
	. "github.com/gnydick/metric-scraper/util"
)

type Emitter interface {
	parseLine(timestamp int64, line string) (*m.Metric, error)
	// Scan fetches the target's metrics and sends them to the sink. The fetch ends when ctx does.
	Scan(ctx context.Context)
	GetName() string
}

// scanClient is the one HTTP client every scan uses. It has its own transport, so a scan sets
// nothing on http.DefaultTransport. It skips certificate checks, as scans always have (#21).
var scanClient = newScanClient()

func newScanClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return &http.Client{Transport: transport}
}

// fetch GETs url and returns the whole body. It ends when ctx does, whether the target is slow to
// answer or slow to finish its body (#21).
func fetch(ctx context.Context, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := scanClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

// logSkipped reports a line a scan left out. A line that is not a metric is ordinary and logs at
// DEBUG; a metric line with a bad value or tag logs at ERROR (#19).
func logSkipped(line string, err error) {
	if errors.Is(err, m.ErrNotAMetric) {
		DebugLog("Skipped a line that is not a metric: %s", line)
		return
	}
	ErrorLog("Skipped a bad metric line: %s", err.Error())
}
