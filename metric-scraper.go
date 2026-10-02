package main

import (
	"fmt"
	"github.com/Unknwon/log"
	"github.com/gorilla/mux"
	"net/http"
	"os"

	c "github.com/gnydick/metric-scraper/config"
	s "github.com/gnydick/metric-scraper/scraper"
	"github.com/gnydick/metric-scraper/telemetry"
	. "github.com/gnydick/metric-scraper/util"
)

func main() {
	scraperPtr, tel, err := startup(os.Getenv("CONFIG_PATH"))
	if err != nil {
		// A reported failure, not a crash (docs/dictated-specs/config.md, Bad config at startup).
		log.Fatal("Startup failed: %s", err.Error())
	}
	router := newRouter(tel)

	go func() {
		_err := http.ListenAndServe(":8765", router)
		if _err != nil {
			log.Fatal("%s", _err.Error())
		}
	}()

	scraperPtr.Scrape()
}

// startup loads and validates the config and builds the scraper and its metrics from it. A bad
// config, or a sink or target that cannot be set up, comes back as an error.
func startup(configPath string) (*s.Scraper, *telemetry.Telemetry, error) {
	config, err := c.FileBuild(configPath)
	if err != nil {
		return nil, nil, err
	}
	if config.Debug() {
		LogLevel = DEBUG
	}
	tel, err := telemetry.New(config.MetricsPrefix())
	if err != nil {
		return nil, nil, err
	}
	scraper, err := s.NewScraper(&config, tel)
	if err != nil {
		return nil, nil, err
	}
	return scraper, tel, nil
}

// newRouter serves /healthz and the metrics page.
func newRouter(tel *telemetry.Telemetry) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/healthz", healthz).Methods("GET")
	router.Handle("/metrics", tel.Handler()).Methods("GET")
	return router
}

// healthz answers OK whenever the process is running. Nothing else changes its answer: the state of
// the sink, of discovery and of the scans is on the metrics page instead
// (docs/dictated-specs/health-and-metrics.md, /healthz).
func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, "OK")
}
