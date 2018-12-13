package emitters

import (
	m "github.com/gnydick/metric-scraper/metric"
	k "github.com/gnydick/metric-scraper/sink"
)

type Emitter interface {
	cleanText (text *string) (string)
	parseLine(timestamp int64, line *string) (m.Metric)
	Scan(sink k.Sink)
}

