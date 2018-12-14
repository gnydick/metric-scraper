package sink

import (
    "fmt"
    c "github.com/gnydick/metric-scraper/config"
    m "github.com/gnydick/metric-scraper/metric"
    op "github.com/gnydick/metric-scraper/output"
    "net"
    "sync"
)

type Opentsdb struct {
    config   c.Config
    receiver *chan *m.Metric
    endpoint string
    wg       *sync.WaitGroup
}

func (o Opentsdb) GetChannel() (*chan *m.Metric) {
    return o.receiver
}

func (o Opentsdb) AddClient() {
    o.wg.Add(1)
}

func (o Opentsdb) RemoveClient() {
    o.wg.Add(-1)
}

func (o Opentsdb) Send() {
    op := op.NewOpentsdbOutput()
    conn, err := net.Dial("tcp", o.endpoint)
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    for metric := range *o.receiver {
        foo := fmt.Sprintf("%s", op.ByteMarshal(metric))
        fmt.Fprintf(conn, foo)
    }
    o.wg.Wait()
    fmt.Println("After wait")
}

func NewOpentsdbSink(config *c.Config, receiverChannel *chan *m.Metric, wg *sync.WaitGroup) (Opentsdb) {
    _, tsdb, _ := net.LookupSRV("", "", config.Metric())
    tsdbAnswer := tsdb[0]
    tsdbEndpoint := fmt.Sprintf("%s:%d", tsdbAnswer.Target, tsdbAnswer.Port)
    if config.Debug() {
        fmt.Println("tsdb endpoint:" + tsdbEndpoint)
    }
    sink := Opentsdb{
        receiver: receiverChannel,
        endpoint: tsdbEndpoint,
        wg:       wg,
    }

    return sink
}
