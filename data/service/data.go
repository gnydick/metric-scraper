package service

import (
    m "github.com/gnydick/metric-scraper/metric"
)

type ServiceData struct {
    metrics []*m.Metric
}

func (s ServiceData) GetMetrics() ([]*m.Metric) {
    return s.metrics
}

func (s *ServiceData) RegisterMetric(metric *m.Metric) {

    s.metrics = append(s.metrics, metric)

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
        metrics: make([]*m.Metric, 0),
    }
    return &ds
}
