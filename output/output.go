package output

import (
	m "github.com/gnydick/metric-scraper/metric"
)

type Output interface {
	StringMarshal(metric m.Metric) (string)
	JsonMarshal(metric m.Metric) ([]byte)
}



