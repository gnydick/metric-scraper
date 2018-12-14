package metric_scraper

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	c "github.com/gnydick/metric-scraper/config"
	s "github.com/gnydick/metric-scraper/scraper"
)

var startTime time.Time

var debug bool
var deploymentId string
var kind string
var disco string
var orch string
var ident string
var interval string

func init() {

	startTime = time.Now()

}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/healthz", GetHealthz).Methods("GET")

	config := c.FileBuild("main/cadvisor.json")

	scraperPtr := s.NewScraper(&config)
	go scraperPtr.Run()

	log.Fatal(http.ListenAndServe(":8765", router))
}

func uptime() time.Duration {
	return time.Since(startTime)
}

func GetHealthz(w http.ResponseWriter, r *http.Request) {
	health := make(map[string]string)
	health["uptime"] = strconv.FormatFloat(uptime().Seconds(), 10, 1, 64)
	health["hostname"], _ = os.Hostname()
	health["metrics_reported"] = strconv.FormatInt(0.0, 10)
	json.NewEncoder(w).Encode(health)
}
