package scraper

import (
    "fmt"

    c "github.com/gnydick/metric-scraper/config"
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
        }
    case "service":
        target := t.NewService(configPtr, "http", sink.(k.Sink))
        scraper = Scraper{
            config:          configPtr,
            metricsReported: 0,
            target:          target,
            sink:            sink.(k.Sink),
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

func (s Scraper) Scrape() {
    DebugLog("Starting scrape")
    d, _ := time.ParseDuration(s.config.Interval())
    go s.sink.Send()
    for {

        x += 1
        DebugLog("go'ing send")

        for _, emitter := range (s.target).EmitterPtrs() {
            DebugLog(fmt.Sprintf("client count before: %d", s.sink.ClientCount()))
            DebugLog(fmt.Sprintf("client count after: %d", s.sink.ClientCount()))
            DebugLog("going scan")
            go emitter.Scan()

        }

        DebugLog("After wait")


        DebugLog("Made it to the end of scrape")
        time.Sleep(d)
    }
}
