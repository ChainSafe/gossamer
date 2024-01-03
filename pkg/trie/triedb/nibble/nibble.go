package nibble

const NibbleLength = 16

type NibbleVec struct {
	inner []byte
	len   uint
}

func (n NibbleVec) RightIter() []byte {
	return n.inner
}

func (n NibbleVec) Len() uint {
	return n.len
}
