package metric

import (
	"log"
	"regexp"
	"strconv"
	"strings"
)


type Service struct {}

func (s Service) Unmarshal(millis int64, line *string) Metric {
	/*
	match  metric_name{tags} value
		first field is metric name
		second
	 */

	metric := Metric{}
	metric.time = millis
	var mnameBytes []byte
	var tagStringBytes []byte
	var valueBytes []byte
	re := regexp.MustCompile(`(?P<metric>[a-z0-9_]+){(?P<tags>[a-z=\",-_]+)} (?P<value>[0-9.+-e]+)`)
	matches := re.FindStringSubmatchIndex(*line)
	if matches != nil {
		metric.metricName = string(re.ExpandString(mnameBytes, "${metric}", *line, matches))
		tagString := string(re.ExpandString(tagStringBytes, "${tags}", *line, matches))

		value, _err := strconv.ParseFloat(string(re.ExpandString(valueBytes, "$value",*line, matches)),64)
		if _err != nil {
			log.Fatal(_err.Error())
		}
		metric.value = value
		tags := strings.Split(tagString, ",")

		mtags := make([]Tag, len(tags))
		for i, tag := range tags {
			tagArray := strings.Split(tag, "=")
			t := Tag {
				key: tagArray[0],
				value: tagArray[1],
			}
			mtags[i] = t


		}
		metric.tags = mtags

	}
	return metric
}
