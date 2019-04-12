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

func (b *branch) header() []byte {
	var header byte
	if b.value == nil {
		header = 2
	} else {
		header = 3
	}
	var encodePkLen []byte

	if len(b.key) > 62 {
		header = header | 0xfc
		encodePkLen = encodeExtraPartialKeyLength(len(b.key))
	} else {
		header = header | (byte(len(b.key)) << 2)
	}

	fullHeader := append([]byte{header}, encodePkLen...)
	return fullHeader
}

func (l *leaf) header() []byte {
	var header byte = 1
	var encodePkLen []byte

	if len(l.key) > 62 {
		header = header | 0xfc
		encodePkLen = encodeExtraPartialKeyLength(len(l.key))
	} else {
		header = header | (byte(len(l.key)) << 2)
	}

	fullHeader := append([]byte{header}, encodePkLen...)
	return fullHeader
}

func encodeExtraPartialKeyLength(pkLen int) []byte {
	pkLen -= 63
	fullHeader := []byte{}
	for i := 0; i < 317; i++ {
		if pkLen < 255 {
			fullHeader = append(fullHeader, byte(pkLen))
			break
		} else {
			fullHeader = append(fullHeader, byte(255))
			pkLen -= 255
		}
	}

	return fullHeader
}
