package scraper

import (
	"context"
	"fmt"
	c "github.com/gnydick/metric-scraper/config"
	"github.com/gnydick/metric-scraper/emitters"
	k "github.com/gnydick/metric-scraper/sink"
	t "github.com/gnydick/metric-scraper/targeting"
	. "github.com/gnydick/metric-scraper/util"
	"sync"
	"time"
)

type Scraper struct {
	config          *c.Config
	metricsReported int64
	target          t.Target
	sink            k.Sink
	emitters        map[string]*emitters.Emitter
	progress        *Progress
}

var x = 0

// NewScraper builds the sink and the target the config names. A sink or target that cannot be set
// up is an error for the caller to report at startup, never a panic (#17).
func NewScraper(configPtr *c.Config) (*Scraper, error) {
	var sink k.Sink
	switch configPtr.Sink() {
	case c.SinkOpentsdb:
		otsdb, err := k.NewOpentsdbSink(configPtr, &sync.WaitGroup{})
		if err != nil {
			return nil, err
		}
		sink = otsdb
	default:
		// A validated Config holds no other sink; this arm is for a sink kind added to the config
		// package and not yet built here.
		return nil, fmt.Errorf("no sink is built for sink kind %q", configPtr.Sink())
	}

	var target t.Target
	switch configPtr.Kind() {
	case c.KindCadvisor:
		cadvisor, err := t.NewCadvisor(configPtr, "http", sink)
		if err != nil {
			return nil, err
		}
		target = cadvisor
	case c.KindService:
		target = t.NewService(configPtr, "http", sink)
	default:
		return nil, fmt.Errorf("no target is built for kind %q", configPtr.Kind())
	}

	return &Scraper{
		config:          configPtr,
		metricsReported: 0,
		target:          target,
		sink:            sink,
		emitters:        make(map[string]*emitters.Emitter),
		progress:        NewProgress(time.Now(), configPtr.Interval()),
	}, nil
}

func (s Scraper) MetricsReported() int64 {
	return s.metricsReported
}

func (s Scraper) IncrMetricsReported() {
	s.metricsReported += 1
}

func (s *Scraper) Scrape() {
	DebugLog("Starting scrape")
	d := s.config.Interval()
	go s.sink.Send()
	for {
		s.scrapeRound(d)
		time.Sleep(d)
	}
}

// Healthy reports whether target discovery succeeded within the last two scrape intervals.
func (s *Scraper) Healthy(now time.Time) bool {
	return s.progress.Healthy(now)
}

// scrapeRound discovers the targets and starts a scan of each. Discovery gets one interval; if it
// fails or runs out of time the round is skipped and the next interval tries again. Each scan's
// fetch also gets one interval, so a target that hangs cannot hold a scan for good (#21).
func (s *Scraper) scrapeRound(interval time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), interval)
	found, err := s.target.EmitterPtrs(ctx)
	cancel()
	if err != nil {
		ErrorLog("Target discovery failed, skipping this round: %s", err.Error())
		return
	}
	s.progress.MarkSuccess(time.Now())

	for _, emitter := range found {
		s.emitters[emitter.GetName()] = &emitter
		go func(emitter emitters.Emitter) {
			scanCtx, cancelScan := context.WithTimeout(context.Background(), interval)
			defer cancelScan()
			emitter.Scan(scanCtx)
		}(emitter)
	}
}
