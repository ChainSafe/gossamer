package trie

type node interface {
	Encode() ([]byte, error)
}

type (
	branch struct {
		key      []byte // partial key
		children [16]node
		value    []byte
	}
	leaf struct {
		key   []byte // partial key
		value []byte
	}
)

func (b *branch) childrenBitmap() uint16 {
	var bitmap uint16
	var i uint
	for i = 0; i < 16; i++ {
		if b.children[i] != nil {
			bitmap = bitmap | 1<<i
		}
	}
	return bitmap
}

func (b *branch) Encode() ([]byte, error) {
	return nil, nil
}

func (l *leaf) Encode() ([]byte, error) {
	return nil, nil
}

func (b *branch) header() (byte) {
	var header byte
	if b.value == nil {
		header = 2
	} else {
		header = 3
	}

	if len(b.key) > 62 {
		header = header | 0xfc
	} else {
		header = header | ((byte(len(b.key)) << 2))
	}

	return header
}

func (l *leaf) header() (byte) {
	var header byte = 1

	if len(l.key) > 62 {
		header = header | 0xfc
	} else {
		header = header | ((byte(len(l.key)) << 2))
	}

	return header	
}