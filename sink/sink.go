package sink

import (
	m "github.com/gnydick/metric-scraper/metric"
)


type Sink interface {
	Send()
	GetChannel() (chan m.Metric)
	AddWg(i int)
	SubWg(i int)
	WaitWg()
}


