package metric

type MetricUnmarshal interface {
	Unmarshal(tags []Tag)
}

type Metric struct {
	metricName    string
	containerName string
	tags          []Tag
	value         float64
	time          int64
}

func (m *Metric) Time() int64 {
	return m.time
}

func (m *Metric) Value() float64 {
	return m.value
}

func (m *Metric) Tags() []Tag {
	return m.tags
}

func (m *Metric) ContainerName() string {
	return m.containerName
}

func (m *Metric) MetricName() string {
	return m.metricName
}

type Tag struct {
	key   string
	value string
}

func (t *Tag) Value() string {
	return t.value
}

func (t *Tag) Key() string {
	return t.key
}


