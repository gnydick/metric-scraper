package sink

import (
	"bufio"
	"fmt"
	"github.com/Unknwon/log"
	c "github.com/gnydick/metric-scraper/config"
	m "github.com/gnydick/metric-scraper/metric"
	op "github.com/gnydick/metric-scraper/output"
	"github.com/gnydick/metric-scraper/telemetry"
	. "github.com/gnydick/metric-scraper/util"
	"io"
	"net"
	"sync"
)

// sinkName is how this sink is labelled on the metrics page.
const sinkName = "opentsdb"

type Opentsdb struct {
	config   c.Config
	receiver *chan *m.Metric
	endpoint string
	wg       *sync.WaitGroup
	clients  int
	// telemetry records the sink's state and writes for the metrics page. It may be nil.
	telemetry *telemetry.Telemetry
}

func (o Opentsdb) ClientCount() int {
	return o.clients
}

func (o Opentsdb) GetChannel() *chan *m.Metric {
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
	op := op.NewOpentsdb()
	conn, err := net.Dial("tcp", o.endpoint)
	if err != nil {
		panic(err)
	}
	// The sink is up from the moment it is connected until the connection is closed or fails.
	o.telemetry.SinkUp(sinkName, o.endpoint, true)
	defer o.telemetry.SinkUp(sinkName, o.endpoint, false)

	// Replies are read on their own goroutine, so a slow or silent OpenTSDB never holds up a put.
	repliesDone := make(chan struct{})
	go func() {
		o.logReplies(conn)
		close(repliesDone)
	}()

	x := 0

	for metric := range *(o.receiver) {
		metricText := fmt.Sprintf("%s", op.StringMarshal(metric))
		// The metric text is data, never a format string: it is written as is (#13).
		_, _err := fmt.Fprint(conn, metricText)
		o.telemetry.SinkWrite(sinkName, _err)

		if hasKey("container_name", getKeys((*metric).Tags)) {
			if (*metric).Tags["container_name"] == "adminserver" {
				DebugLog("%s %s", (*metric).Metric, (*metric).Tags)
				DebugLog("%s", metricText)
			}
		}

		if _err != nil {
			// log.Fatal exits at once, so the deferred call above does not run.
			o.telemetry.SinkUp(sinkName, o.endpoint, false)
			log.Fatal("%s", _err.Error())
		}
		x += 1
	}
	conn.Close()
	// Closing the connection ends the reply reader.
	<-repliesDone
}

// replyLogLimit is the most bytes of one OpenTSDB reply line that are logged.
const replyLogLimit = 1024

// logReplies logs each line OpenTSDB writes back at ERROR and counts it for the metrics page, until
// replies ends. Send only writes puts, and OpenTSDB answers a put only to reject it, so every line
// is a rejected metric (#15).
func (o *Opentsdb) logReplies(replies io.Reader) {
	reader := bufio.NewReaderSize(replies, replyLogLimit)
	midLine := false
	for {
		chunk, isPrefix, err := reader.ReadLine()
		if err != nil {
			return
		}
		// Only the start of a line is logged; the rest of an over-long line is read and dropped.
		if !midLine && len(chunk) > 0 {
			ErrorLog("OpenTSDB rejected a metric: %s", string(chunk))
			o.telemetry.SinkRejection(sinkName, string(chunk))
		}
		midLine = isPrefix
	}
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

// NewOpentsdbSink looks up where OpenTSDB listens. A lookup that fails or finds nothing is an
// error for the caller to report at startup, never a panic (#17).
func NewOpentsdbSink(config *c.Config, wg *sync.WaitGroup, tel *telemetry.Telemetry) (*Opentsdb, error) {

	_, tsdb, err := net.LookupSRV("", "", config.Metric())
	if err != nil {
		return nil, fmt.Errorf("looking up the OpenTSDB address %q: %w", config.Metric(), err)
	}
	if len(tsdb) == 0 {
		return nil, fmt.Errorf("looking up the OpenTSDB address %q: no SRV record", config.Metric())
	}
	tsdbAnswer := tsdb[0]
	tsdbEndpoint := fmt.Sprintf("%s:%d", tsdbAnswer.Target, tsdbAnswer.Port)
	sinkChan := make(chan *m.Metric)
	sink := Opentsdb{
		clients:   0,
		endpoint:  tsdbEndpoint,
		wg:        wg,
		receiver:  &sinkChan,
		telemetry: tel,
	}
	// Known but not connected yet: the gauge is on the page from the start, at 0.
	tel.SinkUp(sinkName, tsdbEndpoint, false)

	return &sink, nil
}
