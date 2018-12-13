package targeting

import (
	"fmt"
	"net"

	c "github.com/gnydick/metric-scraper/config"
	e "github.com/gnydick/metric-scraper/emitters"
	s "github.com/gnydick/metric-scraper/sink"
)

type Service struct {
	config *c.Config
	scheme string
	sink   s.Sink
}

func NewService(config *c.Config, scheme string, sink s.Sink) (Service) {
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

func (s Service) EmitterPtrs() ([]e.Emitter) {
	emitters := make([]e.Emitter, 1)

	emitters[0] = e.NewService(s.sink, s.config, s.assembleServiceEndpoint(), "app="+s.config.Ident())
	return emitters
}

func (s Service) assembleServiceEndpoint() (url string) {
	scrapeTarget := s.config.Disco()
	_, disco, err := net.LookupSRV("", "", scrapeTarget)

	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s://%s:%d/metrics", s.scheme, disco[0].Target, disco[0].Port)
}
