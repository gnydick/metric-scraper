package metric

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrNotAMetric reports a line that is not in the metric line format. It is the ordinary case for
// lines the scraper does not take, so callers log it at DEBUG; any other parse error means a line
// in the format carried bad data.
var ErrNotAMetric = errors.New("line is not a metric")

var taggedRe = regexp.MustCompile(`(?P<metric>[a-z0-9_]+){(?P<tags>[a-z=\",-_]+)} (?P<value>[0-9.+-e]+)`)
var machineRe = regexp.MustCompile(`(?P<metric>machine_[a-z_]+) (?P<value>[0-9.+-e]+)`)

// parseTagged parses "name{key="value",...} value". A line that is not in that shape gives
// ErrNotAMetric. A line in that shape with a bad value or tag gives another error. Either way no
// metric is returned (#19).
func parseTagged(millis int64, line string) (*Metric, error) {
	match := taggedRe.FindStringSubmatch(line)
	if match == nil {
		return nil, ErrNotAMetric
	}
	name, tagString, valueText := match[1], match[2], match[3]

	value, err := parseValue(name, valueText)
	if err != nil {
		return nil, err
	}
	tags := make(map[string]string)
	// A comma after the last tag is legal in the format, so it is dropped before splitting.
	for _, tag := range strings.Split(strings.TrimSuffix(tagString, ","), ",") {
		pair := strings.Split(tag, "=")
		if len(pair) != 2 {
			return nil, fmt.Errorf("metric %s: tag %q is not key=value", name, tag)
		}
		tags[pair[0]] = strings.Trim(pair[1], "\"")
	}
	return &Metric{Metric: name, Tags: tags, Value: value, Time: millis}, nil
}

// parseMachine parses "machine_name value", the untagged lines cadvisor prints for the node.
func parseMachine(millis int64, line string) (*Metric, error) {
	match := machineRe.FindStringSubmatch(line)
	if match == nil {
		return nil, ErrNotAMetric
	}
	name, valueText := match[1], match[2]

	value, err := parseValue(name, valueText)
	if err != nil {
		return nil, err
	}
	return &Metric{Metric: name, Tags: make(map[string]string), Value: value, Time: millis}, nil
}

func parseValue(name string, text string) (float64, error) {
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("metric %s: value %q is not a number", name, text)
	}
	return value, nil
}
