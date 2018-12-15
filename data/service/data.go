package service

import (
    m "github.com/gnydick/metric-scraper/metric"
)

type ServiceData struct {
    metrics map[string]*m.Metric
}

func (s ServiceData) GetMetrics() (*map[string]*m.Metric) {
    return &s.metrics
}

func (s *ServiceData) RegisterMetric(metric *m.Metric) {
    s.metrics[(*metric).Metric] = metric
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
    tags := &(*metric).Tags
    return (*tags)[key]

}



func NewServiceData() *ServiceData {
    mms := make(map[string]*m.Metric)
    ds := ServiceData{
        metrics: mms,
    }
    return &ds
}
