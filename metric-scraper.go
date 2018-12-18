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
    router.HandleFunc("/healthz", GetHealthz).Methods("GET")

    configPath := os.Getenv("CONFIG_PATH")

    config := c.FileBuild(configPath)
    if config.Debug() {
        LogLevel = DEBUG
    }


    DebugLog("Starting up")
    scraperPtr := s.NewScraper(&config)

    go func() {
        _err := http.ListenAndServe(":8765", router)
        if _err != nil {
            log.Fatal(_err.Error())
        }
    }()

    scraperPtr.Scrape()
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
