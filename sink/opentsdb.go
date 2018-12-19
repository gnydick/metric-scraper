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
    return o.receiver
}

func (o Opentsdb) Wait() {
    (*o.wg).Wait()
    close(*o.receiver)
}

func (o *Opentsdb) AddClient() {
    o.clients += 1
    (*o.wg).Add(1)

}

func (o *Opentsdb) RemoveClient() {
    o.clients -= 1
    (*o.wg).Add(-1)
}

func (o *Opentsdb) Send() {
    op := op.NewOpentsdbOutput()
    conn, err := net.Dial("tcp", o.endpoint)
    if err != nil {
        panic(err)
    }

    x := 0

    for metric := range *(o.receiver) {
        metricText := fmt.Sprintf("%s", op.StringMarshal(metric))
        _, _err := fmt.Fprintf(conn, metricText)

        if hasKey("container_name", getKeys((*metric).Tags)) {
            if (*metric).Tags["container_name"] == "adminserver" {
                DebugLog(fmt.Sprintf("%s %s", (*metric).Metric, (*metric).Tags))
                DebugLog(metricText)
            }
        }

        if _err != nil {
            log.Fatal(_err.Error())
        }
        x += 1
    }
    conn.Close()
}

func hasKey(key string, keys []string) bool {
    for _, k := range keys {
        if k == key {
            return true
        }
    }
    return false
}

func getKeys(strings map[string]string) []string {
    var keys = make([]string, len(strings))
    x := 0
    for k, _ := range strings {
        keys[x] = k
        x++
    }
    return keys

}

func NewOpentsdbSink(config *c.Config, wg *sync.WaitGroup) (*Opentsdb) {

    _, tsdb, _ := net.LookupSRV("", "", config.Metric())
    tsdbAnswer := tsdb[0]
    tsdbEndpoint := fmt.Sprintf("%s:%d", tsdbAnswer.Target, tsdbAnswer.Port)
    sinkChan := make(chan *m.Metric)
    sink := Opentsdb{
        clients:  0,
        endpoint: tsdbEndpoint,
        wg:       wg,
        receiver: &sinkChan,
    }

    return &sink
}
