package noop

type Metrics struct{}

func New() (metrics *Metrics) {
	return new(Metrics)
}

func (m *Metrics) NodesAdd(n uint32) {}

func (m *Metrics) NodesSub(n uint32) {}
