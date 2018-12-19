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

func (o *Opentsdb) StringMarshal(metric *m.Metric) string {
    output := fmt.Sprintf("put %s %d %f %s\n", (*metric).Metric,
        (*metric).Time, (*metric).Value, formatTags(metric))

    return cleanText(output)
}

func (o *Opentsdb) ByteMarshal(metric *m.Metric) []byte {
    output, _err := json.MarshalIndent(metric, "", " ")
    if _err != nil {
        log.Fatal(_err.Error())
    }

    text := string(output)
    return []byte(cleanText(text))
}

func formatTags(metric *m.Metric) string {
    tags := (*metric).Tags
    var t = make([]string, len(tags))
    i := 0
    for k, v := range tags {
        t[i] = k + "=" + v
        i++

    }
    return strings.Join(t, " ")
}

func cleanText(text string) (string){
    return strings.Replace(strings.Replace(strings.Replace(strings.Replace(string(*text), `"`, ``, -1), `,`, ` `, -1), `:`, `_`, -1), `@`,`_`, -1)
}

