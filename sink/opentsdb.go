package sink

import (
	"fmt"
	"net"
	"os"
	"sync"

	c "github.com/gnydick/metric-scraper/config"
	m "github.com/gnydick/metric-scraper/metric"
	op "github.com/gnydick/metric-scraper/output"
)

type Opentsdb struct {
	config   c.Config
	receiver chan m.Metric
	endpoint string
	wg       sync.WaitGroup
}

func (o Opentsdb) GetChannel() (chan m.Metric) {
	return o.receiver
}


func (o Opentsdb) WaitWg() {
	o.wg.Wait()
}

func (o Opentsdb) Send() {
	op := op.NewOpentsdbOutput()
	conn, err := net.Dial("tcp", o.endpoint)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	for data := range o.receiver {
		fmt.Fprintf(conn, fmt.Sprintf("%s", op.ByteMartial(data)))
	}

}

func NewOpentsdbSink(config *c.Config, receiverChannel chan m.Metric) (Opentsdb) {
	_, tsdb, _ := net.LookupSRV("", "", os.Getenv("OPENTSDB"))
	tsdbAnswer := tsdb[0]
	tsdbEndpoint := fmt.Sprintf("%s:%d", tsdbAnswer.Target, tsdbAnswer.Port)
	if config.Debug() {
		fmt.Println("tsdb endpoint:" + tsdbEndpoint)
	}
	o := Opentsdb{
		receiver: receiverChannel,
		endpoint: tsdbEndpoint,
	}
	return o
}

func (o Opentsdb) AddWg(i int) {
	o.wg.Add(i)
}

func (o Opentsdb) SubWg(i int) {
	for x := 1; i <= i; x++ {
		o.wg.Done()
	}

}
