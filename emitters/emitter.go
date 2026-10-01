package emitters

import (
	"errors"

	m "github.com/gnydick/metric-scraper/metric"
	. "github.com/gnydick/metric-scraper/util"
)

type Emitter interface {
	parseLine(timestamp int64, line string) (*m.Metric, error)
	Scan()
	GetName() string
}

// logSkipped reports a line a scan left out. A line that is not a metric is ordinary and logs at
// DEBUG; a metric line with a bad value or tag logs at ERROR (#19).
func logSkipped(line string, err error) {
	if errors.Is(err, m.ErrNotAMetric) {
		DebugLog("Skipped a line that is not a metric: %s", line)
		return
	}
	ErrorLog("Skipped a bad metric line: %s", err.Error())
}
