package agent

type Metrics struct {
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{}

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType, networkID string) {
}
