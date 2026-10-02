package emitters

import (
	"bufio"
	"context"
	"strings"
	"time"

	c "github.com/gnydick/metric-scraper/config"
	dataSvc "github.com/gnydick/metric-scraper/data/service"
	m "github.com/gnydick/metric-scraper/metric"
	k "github.com/gnydick/metric-scraper/sink"
	. "github.com/gnydick/metric-scraper/util"
)

type Service struct {
	url         string
	identTag    string
	sink        k.Sink
	serviceData *dataSvc.ServiceData
}

func NewService(sink k.Sink, c *c.Config, url string, identTag string) Service {
	svcData := dataSvc.NewServiceData()

	emitter := Service{
		url:         url,
		identTag:    identTag,
		sink:        sink,
		serviceData: svcData,
	}

	return emitter

}

func (svc Service) parseLine(timestamp int64, line string) (*m.Metric, error) {
	return m.SvcUnmarshal(timestamp, line)
}

func (svc Service) GetName() string {
	return svc.identTag
}

func (svc Service) Scan(ctx context.Context) {
	DebugLog("Starting scan")

	// A failed fetch is logged and the scan ends, as in the cadvisor emitter. It is not a panic (#21).
	body, err := fetch(ctx, svc.url)
	if err != nil {
		ErrorLog("%s", err.Error())
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(body)))

	newMetric := false
	gotType := false
	sinkChan := svc.sink.GetChannel()
	DebugLog("About to scan file")
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
			metric, err := svc.parseLine(millis, line)
			if err != nil {
				logSkipped(line, err)
				continue
			}
			svc.serviceData.RegisterMetric(metric)
		}

	}

	mets := 0

	for _, metric := range svc.serviceData.GetMetrics() {
		mets += 1
		*sinkChan <- metric
	}

}
