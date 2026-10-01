package targeting

import (
	"context"
	"fmt"
	"net"

	c "github.com/gnydick/metric-scraper/config"
	e "github.com/gnydick/metric-scraper/emitters"
	k "github.com/gnydick/metric-scraper/sink"
)

type Service struct {
	config *c.Config
	scheme string
	sink   k.Sink
}

func NewService(config *c.Config, scheme string, sink k.Sink) Service {
	service := Service{
		config: config,
		scheme: scheme,
		sink:   sink,
	}
	return service
}

func (s Service) GetConfig() (config *c.Config) {
	return
}

// EmitterPtrs looks the service up and returns its one emitter. The lookup ends when ctx does, and
// a failed lookup is returned to the caller, which skips the round; it is never panicked (#17).
func (s Service) EmitterPtrs(ctx context.Context) ([]e.Emitter, error) {
	endpoint, err := s.assembleServiceEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	emitters := make([]e.Emitter, 1)

	emitters[0] = e.NewService(s.sink, s.config, endpoint, "app="+s.config.Ident())
	return emitters, nil
}

func (s Service) assembleServiceEndpoint(ctx context.Context) (url string, err error) {
	scrapeTarget := s.config.Disco()
	_, disco, err := net.DefaultResolver.LookupSRV(ctx, "", "", scrapeTarget)
	if err != nil {
		return "", fmt.Errorf("looking up the service address %q: %w", scrapeTarget, err)
	}
	if len(disco) == 0 {
		return "", fmt.Errorf("looking up the service address %q: no SRV record", scrapeTarget)
	}

	return fmt.Sprintf("%s://%s:%d/metrics", s.scheme, disco[0].Target, disco[0].Port), nil
}
