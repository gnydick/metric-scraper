package emitters

import (
	"bufio"
	"context"
	"fmt"
	"k8s.io/api/core/v1"
	"strings"
	"time"

	c "github.com/gnydick/metric-scraper/config"
	dataCadv "github.com/gnydick/metric-scraper/data/cadvisor"
	m "github.com/gnydick/metric-scraper/metric"
	k "github.com/gnydick/metric-scraper/sink"
	. "github.com/gnydick/metric-scraper/util"
)

type Cadvisor struct {
	url    string
	sink   k.Sink
	ds     *dataCadv.DataSet
	node   *v1.Node
	config *c.Config
}

func NewCadvisor(sink k.Sink, c *c.Config, node *v1.Node) Cadvisor {
	ds := dataCadv.NewDataSet(node)
	emitter := Cadvisor{
		url:    fmt.Sprintf("http://%s:%s/metrics/cadvisor", node.Name, "10255"),
		sink:   sink,
		ds:     ds,
		node:   node,
		config: c,
	}

	return emitter

}

func (c Cadvisor) parseLine(timestamp int64, line string) (*m.Metric, error) {
	return m.CadvUnmarshal(timestamp, line)
}

func (c Cadvisor) GetName() string {
	return c.node.Name
}

func (c Cadvisor) Scan(ctx context.Context) {
	DebugLog("Starting scan on %s", c.node.Name)

	body, err := fetch(ctx, c.url)
	if err != nil {
		ErrorLog("%s", err.Error())
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(body)))

	newMetric := false
	gotType := false
	sinkChan := c.sink.GetChannel()
	// DebugLog("About to scan file")
	for scanner.Scan() {
		now := time.Now()
		nanos := now.UnixNano()
		millis := nanos / 1000000
		line := scanner.Text()
		matched := strings.HasPrefix(line, "# HELP ")
		if matched == true {
			gotType = false
			unwanted := strings.HasSuffix(line, "Unix creation timestamp")
			if unwanted == false {
				newMetric = true
			}
		} else if newMetric == true {
			matched := strings.HasPrefix(line, "# TYPE ")
			if matched == true {
				newMetric = false
				gotType = true
			}
		} else if gotType == true {
			metric, err := c.parseLine(millis, line)
			if err != nil {
				logSkipped(line, err)
				continue
			}
			(*metric).Tags["node"] = (*c.node).Name
			c.ds.RegisterMetric(metric)

		}

	}

	var nodes = 0
	for _, node := range *c.ds.GetNodes() {
		nodes += 1
		mets := 0
		// DebugLog(fmt.Sprintf("%d Containers", nodes))
		for _, metric := range *node.GetMetrics() {
			mets += 1
			// DebugLog(fmt.Sprintf("%d metrics", mets))
			*sinkChan <- metric
		}
	}

	conts := 0
	for _, container := range *c.ds.GetContainers() {
		conts += 1
		mets := 0
		// DebugLog(fmt.Sprintf("%d Containers", conts))
		for _, metric := range *container.GetMetrics() {
			mets += 1
			// DebugLog(fmt.Sprintf("%d metrics", mets))
			*sinkChan <- metric
		}
	}
}
