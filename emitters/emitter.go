package emitters

import (
	m "github.com/gnydick/metric-scraper/metric"
)

type Emitter interface {
	parseLine(timestamp int64, line *string) (*m.Metric)
	Scan()
}

