package scraper

import (
	"context"
	"errors"
	"testing"
	"time"

	c "github.com/gnydick/metric-scraper/config"
	"github.com/gnydick/metric-scraper/emitters"
)

// The window is two scrape intervals (#7). With a 10s interval the last moment still healthy is
// 20s after the latest success, or after the start when nothing has succeeded yet.
func TestProgressHealthyWithinTwoIntervals(t *testing.T) {
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	interval := 10 * time.Second

	t.Run("no success yet counts from the start", func(t *testing.T) {
		p := NewProgress(start, interval)
		if !p.Healthy(start.Add(20 * time.Second)) {
			t.Error("unhealthy 20s after the start, want healthy: 20s is within 2 x 10s")
		}
		if p.Healthy(start.Add(20*time.Second + time.Nanosecond)) {
			t.Error("healthy 20s+1ns after the start with no success, want unhealthy")
		}
	})

	t.Run("a success moves the window", func(t *testing.T) {
		p := NewProgress(start, interval)
		p.MarkSuccess(start.Add(30 * time.Second))
		if !p.Healthy(start.Add(50 * time.Second)) {
			t.Error("unhealthy 20s after a success, want healthy")
		}
		if p.Healthy(start.Add(50*time.Second + time.Nanosecond)) {
			t.Error("healthy 20s+1ns after the last success, want unhealthy")
		}
	})
}

// fakeTarget stands in for target discovery. It records the context it was given.
type fakeTarget struct {
	discover func(ctx context.Context) ([]emitters.Emitter, error)
	calls    int
	deadline time.Time
	hadLimit bool
}

func (f *fakeTarget) EmitterPtrs(ctx context.Context) ([]emitters.Emitter, error) {
	f.calls++
	f.deadline, f.hadLimit = ctx.Deadline()
	return f.discover(ctx)
}

func (f *fakeTarget) GetConfig() *c.Config { return nil }

func newTestScraper(target *fakeTarget, start time.Time, interval time.Duration) *Scraper {
	return &Scraper{
		target:   target,
		emitters: make(map[string]*emitters.Emitter),
		progress: NewProgress(start, interval),
	}
}

// A discovery that hangs is cut off after one scrape interval, the round ends, and it does not count
// as progress (#7).
func TestScrapeRoundEndsWhenDiscoveryHangs(t *testing.T) {
	interval := 200 * time.Millisecond
	target := &fakeTarget{discover: func(ctx context.Context) ([]emitters.Emitter, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	start := time.Now()
	s := newTestScraper(target, start, interval)

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
	// The deadline is one interval after the round began, so it lies in (start, now].
	if !target.deadline.After(start) || target.deadline.After(time.Now()) {
		t.Errorf("discovery deadline %v is outside the round (%v, %v]", target.deadline, start, time.Now())
	}
	if s.progress.Healthy(start.Add(2*interval + time.Nanosecond)) {
		t.Error("a round whose discovery timed out was counted as progress")
	}
}

// A failed discovery ends the round without a panic and without counting as progress; a
// successful one counts.
func TestScrapeRoundMarksProgressOnlyOnSuccess(t *testing.T) {
	interval := 10 * time.Second
	// Far enough in the past that only a success marked now can make the scraper healthy now.
	start := time.Now().Add(-time.Hour)

	failing := &fakeTarget{discover: func(ctx context.Context) ([]emitters.Emitter, error) {
		return nil, errors.New("api server said no")
	}}
	s := newTestScraper(failing, start, interval)
	s.scrapeRound(interval)
	if failing.calls != 1 {
		t.Fatalf("discovery ran %d times, want 1", failing.calls)
	}
	if s.progress.Healthy(time.Now()) {
		t.Error("a round whose discovery failed was counted as progress")
	}

	succeeding := &fakeTarget{discover: func(ctx context.Context) ([]emitters.Emitter, error) {
		return []emitters.Emitter{}, nil
	}}
	s = newTestScraper(succeeding, start, interval)
	s.scrapeRound(interval)
	if succeeding.calls != 1 {
		t.Fatalf("discovery ran %d times, want 1", succeeding.calls)
	}
	if !s.progress.Healthy(time.Now()) {
		t.Error("a round whose discovery succeeded was not counted as progress")
	}
}
