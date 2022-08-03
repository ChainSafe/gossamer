package roothash

type Noop struct{}

func NewNoop() (metrics *Noop) {
	return new(Noop)
}

func (n *Noop) NodesAdded(x uint32)   {}
func (n *Noop) NodesDeleted(x uint32) {}
