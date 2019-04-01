package trie

type node interface {
	Encode() ([]byte, error)
	Hash() ([]byte, error)
}

type (
	branch struct {
		children [17]node
	}
	extension struct {
		key   []byte // partial key
		value node   // child node
	}
	leaf struct {
		key   []byte // partial key
		value []byte
	}
)

func (b *branch) childrenBitmap() uint16 {
	var bitmap uint16 = 0
	var i uint
	for i = 0; i < 16; i++ {
		if b.children[i] != nil {
			bitmap = bitmap | 1<<i
		}
	}
	return bitmap
}