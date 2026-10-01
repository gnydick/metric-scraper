package scraper

import (
	"context"
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

func NewScraper(configPtr *c.Config) *Scraper {
	var scraper Scraper

	var sink interface{}
	switch sinkKind := configPtr.Sink(); sinkKind {
	case "opentsdb":
		var otsdb = k.NewOpentsdbSink(configPtr, &sync.WaitGroup{})
		sink = otsdb
	}

	switch kind := configPtr.Kind(); kind {
	case "cadvisor":
		target := t.NewCadvisor(configPtr, "http", sink.(k.Sink))
		scraper = Scraper{
			config:          configPtr,
			metricsReported: 0,
			target:          target,
			sink:            sink.(k.Sink),
			emitters:        make(map[string]*emitters.Emitter),
		}
	case "service":
		target := t.NewService(configPtr, "http", sink.(k.Sink))
		scraper = Scraper{
			config:          configPtr,
			metricsReported: 0,
			target:          target,
			sink:            sink.(k.Sink),
			emitters:        make(map[string]*emitters.Emitter),
		}
	}

	interval, _ := time.ParseDuration(configPtr.Interval())
	scraper.progress = NewProgress(time.Now(), interval)

	return &scraper
}

func (s Scraper) MetricsReported() int64 {
	return s.metricsReported
}

func (s Scraper) IncrMetricsReported() {
	s.metricsReported += 1
}

func (s *Scraper) Scrape() {
	DebugLog("Starting scrape")
	d, _ := time.ParseDuration(s.config.Interval())
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
// fails or runs out of time the round is skipped and the next interval tries again.
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
		go emitter.Scan()
	}
}
