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
	m "github.com/gnydick/metric-scraper/metric"
	k "github.com/gnydick/metric-scraper/sink"
)

type Cadvisor struct {
	url       string
	blacklist []string
	identTag  string
	sink   k.Sink
}

func NewCadvisor(sink k.Sink, c *c.Config, url string, identTag string) (Cadvisor) {
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
	cfgMap := cfg.(map[string]interface{})

	configText := cfgMap["config"].(string)

	emitter := Cadvisor{
		url:       url,
		blacklist: strings.Split(configText, ","),
		identTag:  identTag,
		sink:   sink,
	}

	return emitter

}

func (c Cadvisor) parseLine(timestamp int64, line *string) (m.Metric) {
	cadv := m.Cadvisor{}
	metric := cadv.Unmarshal(timestamp, *line)
	return metric
}

func (c Cadvisor) cleanText(text *string) (string) {
	cleanedText := strings.Replace(strings.Replace(strings.Replace(*text, `"`, ``, -1), `,`, ` `, -1), `:`, `_`, -1)
	return cleanedText
}

func (c Cadvisor) Scan(sink k.Sink) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(c.url)

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	scanner := bufio.NewScanner(strings.NewReader(string(body)))

	newMetric := false
	gotType := false
	sinkChan := sink.GetChannel()
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

			metricPtr := c.parseLine(millis, &line)
			sinkChan <- metricPtr

		}

	}
}
