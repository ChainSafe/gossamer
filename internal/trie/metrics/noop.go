package metrics

var _ Metrics = (*Noop)(nil)

type Noop struct{}

func NewNoop() (metrics *Noop) {
	return new(Noop)
}

func (n *Noop) NodesAdd(x int) {}
