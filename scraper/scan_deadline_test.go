package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gnydick/metric-scraper/emitters"
	m "github.com/gnydick/metric-scraper/metric"
)

// idleSink is a sink nothing reads from.
type idleSink struct {
	metrics chan *m.Metric
}

func (s *idleSink) Send()                       {}
func (s *idleSink) AddClient()                  {}
func (s *idleSink) Wait()                       {}
func (s *idleSink) RemoveClient()               {}
func (s *idleSink) ClientCount() int            { return 0 }
func (s *idleSink) GetChannel() *chan *m.Metric { return &s.metrics }

// A scan the round starts is cut off after one scrape interval (#21). The target accepts the
// request and never answers; the round's scan must give up, which the server sees as its request
// being cancelled.
func TestScrapeRoundCutsAHungScanOffAfterOneInterval(t *testing.T) {
	interval := time.Second

	scanGaveUp := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			scanGaveUp <- struct{}{}
		case <-release:
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	// A scan that is cut off never reaches the sink. The sink is only here so that a scan which is
	// not cut off fails this test cleanly instead of crashing on a missing sink.
	sink := &idleSink{metrics: make(chan *m.Metric, 16)}
	target := &fakeTarget{discover: func(ctx context.Context) ([]emitters.Emitter, error) {
		return []emitters.Emitter{emitters.NewService(sink, nil, srv.URL, "app=x")}, nil
	}}
	s := newTestScraper(target, time.Now(), interval)

	roundBegan := time.Now()
	s.scrapeRound(interval)

	select {
	case <-scanGaveUp:
	case <-time.After(30 * time.Second):
		// Generous on purpose: a scan with no deadline never gives up.
		t.Fatal("the hung scan was still running 30s into a 1s interval")
	}

	// The bound is one interval: the scan cannot give up sooner, and a bound of two intervals or
	// more would give up later than this. The upper limit leaves one whole interval of slack for a
	// slow machine.
	heldFor := time.Since(roundBegan)
	if heldFor < interval || heldFor >= 2*interval {
		t.Errorf("the hung scan gave up after %v, want at least %v and less than %v", heldFor, interval, 2*interval)
	}
}
