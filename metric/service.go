package metric

type Service struct{}

// SvcUnmarshal parses one line of a service's metrics output: a tagged metric line.
func SvcUnmarshal(millis int64, line string) (*Metric, error) {
	return parseTagged(millis, line)
}
