package service

import (
    "fmt"
    m "github.com/gnydick/metric-scraper/metric"
    . "github.com/gnydick/metric-scraper/util"
)

type ServiceData struct {
    metrics []*m.Metric
}

func (s ServiceData) GetMetrics() ([]*m.Metric) {
    return s.metrics
}

func (s *ServiceData) RegisterMetric(metric *m.Metric) {
    for key, _ := range (*metric).Tags {
        if key == "container" {
            DebugLog(fmt.Sprintf("BEFORE CHANNEL %s ",metric))
        }
    }
    s.metrics = append(s.metrics, metric)
    for key, _ := range (*metric).Tags {
        if key == "container" {
            DebugLog(fmt.Sprintf("DURING REGISTER %s ", *metric))
        }
    }

}

func (s ServiceData) hasTagKey(key string, metric *m.Metric) bool {
    for k := range (*metric).Tags {
        if k == key {
            return true
        }
    }
    return false
}

func (s ServiceData) getTagValue(key string, metric *m.Metric) string {
    tags := (*metric).Tags
    return tags[key]

}



func NewServiceData() *ServiceData {
    ds := ServiceData{
        metrics: make([]*m.Metric,0),
    }
    return &ds
}
