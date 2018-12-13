package targeting

import (
	c "github.com/gnydick/metric-scraper/config"
	e "github.com/gnydick/metric-scraper/emitters"
	k "github.com/gnydick/metric-scraper/sink"
)

type Target interface {
	EmitterPtrs() ([]e.Emitter)
	GetConfig() (config *c.Config)
}

type Targeter struct {
	configPtr *c.Config
	scheme    string
	sinkPtr   *k.Sink
}
