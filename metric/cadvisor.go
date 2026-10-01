package metric

import "errors"

type Cadvisor struct {
}

// CadvUnmarshal parses one line of cadvisor output: a tagged metric line, or an untagged
// machine_ line.
func CadvUnmarshal(millis int64, line string) (*Metric, error) {
	metric, err := parseTagged(millis, line)
	if errors.Is(err, ErrNotAMetric) {
		return parseMachine(millis, line)
	}
	return metric, err
}
