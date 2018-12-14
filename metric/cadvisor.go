package metric

import (
    "log"
    "regexp"
    "strconv"
    "strings"
)

type Cadvisor struct{}

func (c Cadvisor) Unmarshal(millis int64, line string) Metric {
    /*
    match  metric_name{tags} value
        first field is Metric name
        second
     */

    metric := Metric{}
    metric.Time = millis
    var mnameBytes []byte
    var tagStringBytes []byte
    var valueBytes []byte
    re := regexp.MustCompile(`(?P<metric>[a-z0-9_]+){(?P<tags>[a-z=\",-_]+)} (?P<value>[0-9.+-e]+)`)
    matches := re.FindStringSubmatchIndex(line)
    if matches != nil {
        metric.Metric = string(re.ExpandString(mnameBytes, "${metric}", line, matches))
        tagString := string(re.ExpandString(tagStringBytes, "${tags}", line, matches))

        value, _err := strconv.ParseFloat(string(re.ExpandString(valueBytes, "$value", line, matches)), 64)
        if _err != nil {
            log.Fatal(_err.Error())
        }
        metric.Value = value
        tags := strings.Split(tagString, ",")

        mtags := make(map[string]string)
        for _, tag := range tags {
            tagArray := strings.Split(tag, "=")
            if len(tagArray) == 2 {
                mtags[tagArray[0]] = strings.Trim(tagArray[1], "\"")
            }

        }
        metric.Tags = mtags

    }
    return metric
}
