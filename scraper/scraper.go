package scraper

import (
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
}

var x = 0

func NewScraper(configPtr *c.Config) (*Scraper) {
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

        for _, emitter := range (s.target).EmitterPtrs() {
            s.emitters[emitter.GetName()] = &emitter
            go emitter.Scan()

        }

        time.Sleep(d)
    }
}
