package emitters

import (
    "bufio"
    "crypto/tls"
    "fmt"
    "io/ioutil"

    "net/http"
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

func NewService(sink k.Sink, c *c.Config, url string, identTag string) (Service) {
    svcData := dataSvc.NewServiceData()

    emitter := Service{
        url:         url,
        identTag:    identTag,
        sink:        sink,
        serviceData: svcData,
    }

    return emitter

}

func (svc Service) parseLine(timestamp int64, line *string) (*m.Metric) {

    metric := m.SvcUnmarshal(timestamp, line)

    return metric
}

func (svc Service) GetName() string {
    return svc.identTag
}

func (svc Service) Scan() {
    DebugLog("Starting scan")

    http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    resp, err := http.Get(svc.url)

    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)

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
            metric := svc.parseLine(millis, &line)
            svc.serviceData.RegisterMetric(metric)
            for key, _ := range (*metric).Tags {
                if key == "container" {
                    DebugLog(fmt.Sprintf("RIGHT BEFORE SINK %s ", *metric))
                }
            }
        }

    }

    mets := 0

    for _, metric := range svc.serviceData.GetMetrics() {
        mets += 1
        for key, _ := range (*metric).Tags {
            if key == "container" {
                DebugLog(fmt.Sprintf("RIGHT BEFORE SINK %s ", *metric))
            }
        }
        *sinkChan <- metric
    }

    DebugLog(fmt.Sprintf("%s", svc.serviceData))
    DebugLog("Releasing Channel")
    DebugLog(fmt.Sprintf("client count before: %d", svc.sink.ClientCount()))
    DebugLog("ending scan")

    // c.sink.RemoveClient()
    DebugLog(fmt.Sprintf("client count after: %d", svc.sink.ClientCount()))
    DebugLog("Removed client")

}
