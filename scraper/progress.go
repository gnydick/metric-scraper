package scraper

import (
	"sync/atomic"
	"time"
)

// Progress records when target discovery last succeeded, so /healthz can tell a scraper that is
// working from one that is stalled (#7). The scrape loop writes it and the HTTP handler reads it.
type Progress struct {
	window time.Duration
	// last is the latest success as Unix nanoseconds. Until the first success it holds the start.
	last atomic.Int64
}

// NewProgress starts the clock at start. The scraper counts as healthy for two intervals after its
// latest success, or after start when nothing has succeeded yet.
func NewProgress(start time.Time, interval time.Duration) *Progress {
	p := &Progress{window: 2 * interval}
	p.last.Store(start.UnixNano())
	return p
}

func (p *Progress) MarkSuccess(at time.Time) {
	p.last.Store(at.UnixNano())
}

func (p *Progress) Healthy(now time.Time) bool {
	return now.Sub(time.Unix(0, p.last.Load())) <= p.window
}
