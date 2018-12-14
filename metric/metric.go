package metric

type Metric struct {
    Metric string            `json:"metric,omitempty"`
    Tags   map[string]string `json:"tags,omitempty"`
    Value  float64           `json:"value"`
    Time   int64             `json:"timestamp,omitempty"`
}
