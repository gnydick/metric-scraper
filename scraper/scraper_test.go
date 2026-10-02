package scraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	c "github.com/gnydick/metric-scraper/config"
	"github.com/gnydick/metric-scraper/emitters"
	"github.com/gnydick/metric-scraper/telemetry"
	"github.com/gnydick/metric-scraper/util/testsupport"
)

// fakeTarget stands in for target discovery. It records the context it was given.
type fakeTarget struct {
	discover func(ctx context.Context) ([]emitters.Emitter, error)
	calls    int
	calledAt time.Time
	deadline time.Time
	hadLimit bool
}

func (f *fakeTarget) EmitterPtrs(ctx context.Context) ([]emitters.Emitter, error) {
	f.calls++
	f.calledAt = time.Now()
	f.deadline, f.hadLimit = ctx.Deadline()
	return f.discover(ctx)
}

func (f *fakeTarget) GetConfig() *c.Config { return nil }

// newTestScraper builds a scraper of kind "service" around target, recording into its own metrics
// page.
func newTestScraper(t *testing.T, target *fakeTarget) *Scraper {
	t.Helper()
	tel, err := telemetry.New("scraper")
	if err != nil {
		t.Fatal(err)
	}
	return &Scraper{
		target:    target,
		kind:      "service",
		emitters:  make(map[string]*emitters.Emitter),
		telemetry: tel,
	}
}

func metricsPage(t *testing.T, s *Scraper) string {
	t.Helper()
	return testsupport.Page(t, s.telemetry.Handler(), "/metrics")
}

// A discovery that hangs is cut off after one scrape interval, the round ends, and it is counted as
// a failed discovery round (#7, #53).
func TestScrapeRoundEndsWhenDiscoveryHangs(t *testing.T) {
	interval := 200 * time.Millisecond
	target := &fakeTarget{discover: func(ctx context.Context) ([]emitters.Emitter, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	start := time.Now()
	s := newTestScraper(t, target)

	done := make(chan struct{})
	go func() {
		s.scrapeRound(interval)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		// Generous on purpose: this only catches the hang, it is not the bound under test.
		t.Fatal("scrapeRound still blocked 30s into a 200ms interval")
	}

	// The observer is alive only if discovery was actually asked.
	if target.calls != 1 {
		t.Fatalf("discovery ran %d times, want 1", target.calls)
	}
	if !target.hadLimit {
		t.Fatal("discovery got a context with no deadline")
	}
	// The deadline is one interval after the context was made. That moment lies between the start
	// of the test and the call into discovery, so the deadline lies one interval after each.
	earliest, latest := start.Add(interval), target.calledAt.Add(interval)
	if target.deadline.Before(earliest) || target.deadline.After(latest) {
		t.Errorf("discovery deadline %v is not one interval (%v) into the round: want within [%v, %v]",
			target.deadline, interval, earliest, latest)
	}
	want := `scraper_discovery_rounds_total{kind="service",result="error"} 1`
	if page := metricsPage(t, s); !testsupport.HasLine(page, want) {
		t.Errorf("the metrics page has no line %q after a discovery that timed out", want)
	}
}

// A failed discovery ends the round without a panic. Each round is counted by its result: one
// failed round and two good rounds give 1 and 2 (#53).
func TestScrapeRoundCountsDiscoveryByResult(t *testing.T) {
	interval := 10 * time.Second
	fail := true
	target := &fakeTarget{discover: func(ctx context.Context) ([]emitters.Emitter, error) {
		if fail {
			return nil, errors.New("api server said no")
		}
		return []emitters.Emitter{}, nil
	}}
	s := newTestScraper(t, target)

	s.scrapeRound(interval)
	fail = false
	s.scrapeRound(interval)
	s.scrapeRound(interval)

	if target.calls != 3 {
		t.Fatalf("discovery ran %d times, want 3", target.calls)
	}
	page := metricsPage(t, s)
	for _, want := range []string{
		`scraper_discovery_rounds_total{kind="service",result="error"} 1`,
		`scraper_discovery_rounds_total{kind="service",result="ok"} 2`,
	} {
		if !testsupport.HasLine(page, want) {
			t.Errorf("the metrics page has no line %q", want)
		}
	}
}

// A target that is no longer discovered leaves the metrics page in the round that misses it (#53).
// Round one finds a and b; round two finds only a.
func TestScrapeRoundDropsATargetThatIsNoLongerDiscovered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "")
	}))
	defer srv.Close()

	sink := &idleSink{}
	names := []string{"app=a", "app=b"}
	var s *Scraper
	target := &fakeTarget{discover: func(ctx context.Context) ([]emitters.Emitter, error) {
		var found []emitters.Emitter
		for _, name := range names {
			found = append(found, emitters.NewService(sink, nil, srv.URL, name, s.telemetry))
		}
		return found, nil
	}}
	s = newTestScraper(t, target)
	interval := 10 * time.Second
	lineFor := func(name string) string {
		return `scraper_target_up{kind="service",target="` + name + `"} 1`
	}

	s.scrapeRound(interval)
	// The scans run on their own goroutines; wait until both have reported.
	deadline := time.Now().Add(30 * time.Second)
	for {
		page := metricsPage(t, s)
		if testsupport.HasLine(page, lineFor("app=a")) && testsupport.HasLine(page, lineFor("app=b")) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the first round's two scans had not reported within 30s")
		}
		time.Sleep(10 * time.Millisecond)
	}

	names = []string{"app=a"}
	s.scrapeRound(interval)

	page := metricsPage(t, s)
	if testsupport.HasLine(page, lineFor("app=b")) {
		t.Error("app=b is still on the metrics page after a round that did not discover it")
	}
	if !testsupport.HasLine(page, lineFor("app=a")) {
		t.Error("app=a left the metrics page although it is still discovered")
	}
}
