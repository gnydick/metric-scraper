package metric

import (
	"log"
	"regexp"
	"strconv"
	"strings"
)

var svcRe = regexp.MustCompile(`(?P<Metric>[a-z0-9_]+){(?P<tags>[a-z=\",-_]+)} (?P<value>[0-9.+-e]+)`)
type Service struct {}

func SvcUnmarshal(millis int64, line *string) *Metric {
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

	matches := svcRe.FindStringSubmatchIndex(*line)
	if matches != nil {
		metric.Metric = string(svcRe.ExpandString(mnameBytes, "${Metric}", *line, matches))
		tagString := string(svcRe.ExpandString(tagStringBytes, "${tags}", *line, matches))

		value, _err := strconv.ParseFloat(string(svcRe.ExpandString(valueBytes, "$value",*line, matches)),64)
		if _err != nil {
			log.Fatal(_err.Error())
		}
		metric.Value = value
		tags := strings.Split(tagString, ",")

		mtags := make(map[string]string)
		for _, tag := range tags {
			tagArray := strings.Split(tag, "=")

            mtags[tagArray[0]] = strings.Trim(tagArray[1], "\"")


		}
		metric.Tags = mtags

	}
	return &metric
}
