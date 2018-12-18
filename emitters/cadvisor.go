package emitters

import (
    "bufio"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "strings"
    "time"

    c "github.com/gnydick/metric-scraper/config"
    dataCadv "github.com/gnydick/metric-scraper/data/cadvisor"
    m "github.com/gnydick/metric-scraper/metric"
    k "github.com/gnydick/metric-scraper/sink"
    . "github.com/gnydick/metric-scraper/util"
)

type Cadvisor struct {
    node      string
    url       string
    blacklist []string
    identTag  string
    sink      k.Sink
    ds        *dataCadv.DataSet
}

func NewCadvisor(node string, sink k.Sink, c *c.Config, url string, identTag string) (Cadvisor) {
    ds := dataCadv.NewDataSet()
    orchUrl := fmt.Sprintf("http://%s/api/rest/v1/config/%s:%s", c.Orch(), c.DeploymentId(), "scraper_tag_blacklist:tag_key_blacklist:default")
    resp, err := http.Get(orchUrl)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    var cfg interface{}

    config := json.RawMessage{}
    json.NewDecoder(resp.Body).Decode(&config)
    err = json.Unmarshal(config, &cfg)
    if err != nil {
        panic(err)
    }
    // cfgMap := cfg.(map[string]interface{})

    // configText := cfgMap["config"].(string)

    emitter := Cadvisor{
        url:       url,
        identTag:  identTag,
        sink:      sink,
        ds:        ds,
        node:      node,
    }

    return emitter

}

func (c Cadvisor) parseLine(timestamp int64, line *string) (*m.Metric) {

    metric := m.CadvUnmarshal(timestamp, line)
    (*metric).Tags["node"] = c.node
    return metric
}

func (e Cadvisor) Scan() {
    DebugLog("Starting scan")

    http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    resp, err := http.Get(e.url)

    if err != nil {
    	panic(err)
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)

    scanner := bufio.NewScanner(strings.NewReader(string(body)))


    newMetric := false
    gotType := false
    sinkChan := e.sink.GetChannel()
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
            e.ds.RegisterMetric(e.parseLine(millis, &line))

        }

    }
    conts := 0
    for _, container := range *e.ds.GetContainers() {
        conts += 1
        mets := 0
        DebugLog(fmt.Sprintf("%d Containers", conts))
        for _, metric := range *container.GetMetrics() {
            mets += 1
            DebugLog(fmt.Sprintf("%d Metrics", mets))
            *sinkChan <- metric
        }
    }

    DebugLog(fmt.Sprintf("%s", e.ds))
    DebugLog("Releasing Channel")
    DebugLog(fmt.Sprintf("client count before: %d", e.sink.ClientCount()))
    DebugLog("ending scan")

    // c.sink.RemoveClient()
    DebugLog(fmt.Sprintf("client count after: %d", e.sink.ClientCount()))
    DebugLog("Removed client")

}
