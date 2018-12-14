package emitters

import (
    "bufio"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "strings"
    "time"

    c "github.com/gnydick/metric-scraper/config"
    dataCadv "github.com/gnydick/metric-scraper/data/cadvisor"
    m "github.com/gnydick/metric-scraper/metric"
    k "github.com/gnydick/metric-scraper/sink"
)

type Cadvisor struct {
    url       string
    blacklist []string
    identTag  string
    sink      k.Sink
    ds        *dataCadv.DataSet
}

func NewCadvisor(sink k.Sink, c *c.Config, url string, identTag string) (Cadvisor) {
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
    //
    // configText := cfgMap["config"].(string)

    emitter := Cadvisor{
        url:       url,
        blacklist: make([]string, 0),
        identTag:  identTag,
        sink:      sink,
        ds:         ds,
    }

    return emitter

}

func (c Cadvisor) parseLine(timestamp int64, line *string) (*m.Metric) {
    // cleanedText := c.cleanText(line)
    cadv := m.Cadvisor{}
    metric := cadv.Unmarshal(timestamp, *line)
    return &metric
}

func (c Cadvisor) cleanText(text *string) (string) {
    cleanedText := strings.Replace(strings.Replace(strings.Replace(*text, `"`, ``, -1), `,`, ` `, -1), `:`, `_`, -1)
    return cleanedText
}

func (c Cadvisor) Scan() {
    c.sink.AddClient()

    //
    // http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    // resp, err := http.Get(c.url)
    //
    // if err != nil {
    // 	panic(err)
    // }
    // defer resp.Body.Close()
    // body, err := ioutil.ReadAll(resp.Body)
    //
    // scanner := bufio.NewScanner(strings.NewReader(string(body)))
    file, _ := os.Open("data/cadvisor.txt")

    scanner := bufio.NewScanner(bufio.NewReader(file))
    newMetric := false
    gotType := false
    sinkChan := c.sink.GetChannel()
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
            c.ds.RegisterMetric(c.parseLine(millis, &line))

        }

    }
    for _, container := range *c.ds.GetContainers() {
        for _, metric := range *container.GetMetrics() {
            *sinkChan <- metric
        }
    }
    close(*sinkChan)
    fmt.Println(c.ds)
    c.sink.RemoveClient()
    fmt.Println("Removed client")

}
