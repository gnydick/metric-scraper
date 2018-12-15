package sink

import (
	m "github.com/gnydick/metric-scraper/metric"
)


type Sink interface {
	Send()
    AddClient()
	Wait()
	RemoveClient()
	GetChannel() (*chan *m.Metric)
	ClientCount() int
}


