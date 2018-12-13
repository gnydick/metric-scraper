package output

import (
	m "github.com/gnydick/metric-scraper/metric"
)

type Output interface {
	StringMarshal(metric m.Metric) (string)
	ByteMarshal(metric m.Metric) ([]byte)
}



