package targeting

import (
	"context"

	c "github.com/gnydick/metric-scraper/config"
	e "github.com/gnydick/metric-scraper/emitters"
	k "github.com/gnydick/metric-scraper/sink"
)

type Target interface {
	// EmitterPtrs discovers the scrape targets. It ends when ctx does and reports a failed discovery.
	EmitterPtrs(ctx context.Context) ([]e.Emitter, error)
	GetConfig() (config *c.Config)
}

type Targeter struct {
	configPtr *c.Config
	scheme    string
	sinkPtr   *k.Sink
}
