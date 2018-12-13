package output

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	m "github.com/gnydick/metric-scraper/metric"
)

type Opentsdb struct {
}

func NewOpentsdbOutput() *Opentsdb {

	return &Opentsdb{}
}

func (o *Opentsdb) StringMarshal(metric m.Metric) string {
	output := fmt.Sprintf("put %s %d %f %s", metric.MetricName(),
		metric.Time(), metric.Value(), formatTags(metric.Tags()))
	return output
}

func (o *Opentsdb) ByteMartial(metric m.Metric) []byte {
	output, _err := json.Marshal(metric)
	if _err != nil {
		log.Fatal(_err.Error())
	}
	return output
}

func formatTags(tags []m.Tag) string {
	var t = make([]string, len(tags))
	for i, tag := range tags {
		t[i] = tag.Key() + "=" + tag.Value()
	}
	return strings.Join(t, " ")
}
