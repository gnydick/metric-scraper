package main

import (
	"encoding/json"
	"github.com/Unknwon/log"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"strconv"
	"time"

	c "github.com/gnydick/metric-scraper/config"
	s "github.com/gnydick/metric-scraper/scraper"
	. "github.com/gnydick/metric-scraper/util"
)

var startTime time.Time

func init() {

	startTime = time.Now()

}

func main() {

	router := mux.NewRouter()

	scraperPtr, err := startup(os.Getenv("CONFIG_PATH"))
	if err != nil {
		// A reported failure, not a crash (docs/dictated-specs/config.md, Bad config at startup).
		log.Fatal("Startup failed: %s", err.Error())
	}
	router.HandleFunc("/healthz", healthzHandler(func() bool {
		return scraperPtr.Healthy(time.Now())
	})).Methods("GET")

	go func() {
		_err := http.ListenAndServe(":8765", router)
		if _err != nil {
			log.Fatal("%s", _err.Error())
		}
	}()

	scraperPtr.Scrape()
}

// startup loads and validates the config and builds the scraper from it. A bad config, or a sink
// or target that cannot be set up, comes back as an error.
func startup(configPath string) (*s.Scraper, error) {
	config, err := c.FileBuild(configPath)
	if err != nil {
		return nil, err
	}
	if config.Debug() {
		LogLevel = DEBUG
	}
	return s.NewScraper(&config)
}

func uptime() time.Duration {
	return time.Since(startTime)
}

// healthzHandler answers 200 while healthy() holds and 503 otherwise, with the same JSON report.
func healthzHandler(healthy func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := make(map[string]string)
		health["uptime"] = strconv.FormatFloat(uptime().Seconds(), 10, 1, 64)
		health["hostname"], _ = os.Hostname()
		health["metrics_reported"] = strconv.FormatInt(0.0, 10)
		if !healthy() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(health)
	}
}
