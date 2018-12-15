package sink

import (
    "fmt"
    "github.com/Unknwon/log"
    c "github.com/gnydick/metric-scraper/config"
    m "github.com/gnydick/metric-scraper/metric"
    op "github.com/gnydick/metric-scraper/output"
    . "github.com/gnydick/metric-scraper/util"
    "net"
    "sync"
)

type Opentsdb struct {
    config   c.Config
    receiver *chan *m.Metric
    endpoint string
    wg       *sync.WaitGroup
    clients  int
}

func (o Opentsdb) ClientCount() int {
    return o.clients
}

func (o Opentsdb) GetChannel() (*chan *m.Metric) {
    DebugLog("About to return channel")
    return o.receiver
}

func (o Opentsdb) Wait() {
    (*o.wg).Wait()
    close(*o.receiver)
    DebugLog("After wait")
}

func (o *Opentsdb) AddClient() {
    o.clients += 1
    (*o.wg).Add(1)
    DebugLog("Added client")
}

func (o *Opentsdb) RemoveClient() {
    o.clients -= 1
    (*o.wg).Add(-1)
}

func (o *Opentsdb) Send() {
    DebugLog("Starting Send")
    op := op.NewOpentsdbOutput()
    conn, err := net.Dial("tcp", o.endpoint)
    if err != nil {
        panic(err)
    }

    x := 0
    DebugLog("About to range channel")

    for metric := range *(o.receiver) {

        metricText := fmt.Sprintf("%s", op.StringMarshal(metric))
        _, _err := fmt.Fprintf(conn, metricText)
        DebugLog(metricText)
        if _err != nil {
            log.Fatal(_err.Error())
        }
        x += 1
        DebugLog(fmt.Sprintf("Sent %d messages", x))
    }
    conn.Close()
}

func NewOpentsdbSink(config *c.Config, wg *sync.WaitGroup) (*Opentsdb) {

    _, tsdb, _ := net.LookupSRV("", "", config.Metric())
    tsdbAnswer := tsdb[0]
    tsdbEndpoint := fmt.Sprintf("%s:%d", tsdbAnswer.Target, tsdbAnswer.Port)
    if config.Debug() {
        DebugLog("tsdb endpoint:" + tsdbEndpoint)
    }
    sinkChan := make(chan *m.Metric)
    sink := Opentsdb{
        clients:  0,
        endpoint: tsdbEndpoint,
        wg:       wg,
        receiver: &sinkChan,
    }

    return &sink
}
