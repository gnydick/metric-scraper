package sink

import (
	m "github.com/gnydick/metric-scraper/metric"
)


type Sink interface {
	Send()
	GetChannel() (*chan *m.Metric)
	AddClient()
	RemoveClient()
}


