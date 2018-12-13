package scraper

import (
	"time"

	c "github.com/gnydick/metric-scraper/config"
	m "github.com/gnydick/metric-scraper/metric"
	k "github.com/gnydick/metric-scraper/sink"
	t "github.com/gnydick/metric-scraper/targeting"
)

type Scraper struct {
	config          *c.Config
	metricsReported int64
	target          t.Target
	sink            k.Sink
}

func NewScraper(configPtr *c.Config) (*Scraper) {
	sinkChan := make(chan m.Metric)
	var scraper Scraper
	var sink k.Sink
	switch sinkKind := configPtr.Sink(); sinkKind {
	case "opentsdb":
		sink = k.NewOpentsdbSink(configPtr, sinkChan)
	}

	// var _ t.Target = (*t.Cadvisor)(nil)

	switch kind := configPtr.Kind(); kind {
	case "cadvisor":
		target := t.NewCadvisor(configPtr, "http", sink)
		scraper = Scraper{
			config:          configPtr,
			metricsReported: 0,
			target:          target,
			sink:            sink,
		}
	case "service":
		target := t.NewService(configPtr, "http", sink)
		scraper = Scraper{
			config:          configPtr,
			metricsReported: 0,
			target:          target,
			sink:            sink,
		}
	}


	return &scraper
}

func (s *Scraper) MetricsReported() int64 {
	return s.metricsReported
}

func (s *Scraper) IncrMetricsReported() {
	s.metricsReported += 1
}

func (s *Scraper) ScrapeRoutine(targetPtr *t.Target) {
	var d time.Duration
	d, _ = time.ParseDuration(s.config.Interval())
	s.scrape(targetPtr)
	for {
		time.Sleep(d)
		s.scrape(targetPtr)
	}
}

func (s *Scraper) Run() {
	s.ScrapeRoutine(&s.target)

}

func (s *Scraper) scrape(targetPtr *t.Target) {
	var sink k.Sink
	switch sinkKind := s.config.Sink(); sinkKind {
	case "opentsdb":
		sink = k.NewOpentsdbSink(s.config, s.sink.GetChannel())
		go sink.Send()
	}

	for _, emitter := range (*targetPtr).EmitterPtrs(){
		sink.AddWg(1)
		go (emitter).Scan(sink)
	}
	sink.WaitWg()
	close(sink.GetChannel())

}

