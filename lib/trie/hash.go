package trie

import (
	"bytes"
	"context"
	"hash"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/sync/errgroup"
)

// Hasher is a wrapper around a hash function
type hasher struct {
	hash     hash.Hash
	tmp      bytes.Buffer
	parallel bool // Whether to use parallel threads when hashing
}

// hasherPool creates a pool of Hasher.
var hasherPool = sync.Pool{
	New: func() interface{} {
		h, _ := blake2b.New256(nil)
		var buf bytes.Buffer
		// This allocation will be helpful for encoding keys. This is the min buffer size.
		buf.Grow(700)

		return &hasher{
			tmp:  buf,
			hash: h,
		}
	},
}

// NewHasher create new Hasher instance
func newHasher(parallel bool) *hasher {
	h := hasherPool.Get().(*hasher)
	h.parallel = parallel
	return h
}

func (h *hasher) returnToPool() {
	h.tmp.Reset()
	h.hash.Reset()
	hasherPool.Put(h)
}

// Hash encodes the node and then hashes it if its encoded length is > 32 bytes
func (h *hasher) Hash(n node) (res []byte, err error) {
	encNode, err := h.encode(n)
	if err != nil {
		return nil, err
	}

	// if length of encoded leaf is less than 32 bytes, do not hash
	if len(encNode) < 32 {
		return encNode, nil
	}

	h.hash.Reset()
	// otherwise, hash encoded node
	_, err = h.hash.Write(encNode)
	if err == nil {
		res = h.hash.Sum(nil)
	}

	return res, err
}

// encode is the high-level function wrapping the encoding for different node types
// encoding has the following format:
// NodeHeader | Extra partial key length | Partial Key | Value
func (h *hasher) encode(n node) ([]byte, error) {
	switch n := n.(type) {
	case *branch:
		return h.encodeBranch(n)
	case *leaf:
		return h.encodeLeaf(n)
	case nil:
		return []byte{0}, nil
	}

	return nil, nil
}

func encodeAndHash(n node) ([]byte, error) {
	h := newHasher(false)
	defer h.returnToPool()

	encChild, err := h.Hash(n)
	if err != nil {
		return nil, err
	}

	scEncChild, err := scale.Marshal(encChild)
	if err != nil {
		return nil, err
	}
	return scEncChild, nil
}

// encodeBranch encodes a branch with the encoding specified at the top of this package
func (h *hasher) encodeBranch(b *branch) ([]byte, error) {
	if !b.dirty && b.encoding != nil {
		return b.encoding, nil
	}
	h.tmp.Reset()

	encoding, err := b.header()
	h.tmp.Write(encoding)
	if err != nil {
		return nil, err
	}

	h.tmp.Write(nibblesToKeyLE(b.key))
	h.tmp.Write(common.Uint16ToBytes(b.childrenBitmap()))

	if b.value != nil {
		bytes, err := scale.Marshal(b.value)
		if err != nil {
			return nil, err
		}
		h.tmp.Write(bytes)
	}

	if h.parallel {
		wg, _ := errgroup.WithContext(context.Background())
		resBuff := make([][]byte, 16)
		for i := 0; i < 16; i++ {
			func(i int) {
				wg.Go(func() error {
					child := b.children[i]
					if child == nil {
						return nil
					}

					var err error
					resBuff[i], err = encodeAndHash(child)
					if err != nil {
						return err
					}
					return nil
				})
			}(i)
		}
		if err := wg.Wait(); err != nil {
			return nil, err
		}

		for _, v := range resBuff {
			if v != nil {
				h.tmp.Write(v)
			}
		}
	} else {
		for i := 0; i < 16; i++ {
			if child := b.children[i]; child != nil {
				scEncChild, err := encodeAndHash(child)
				if err != nil {
					return nil, err
				}
				h.tmp.Write(scEncChild)
			}
		}
	}

	return h.tmp.Bytes(), nil
}

// encodeLeaf encodes a leaf with the encoding specified at the top of this package
func (h *hasher) encodeLeaf(l *leaf) ([]byte, error) {
	if !l.dirty && l.encoding != nil {
		return l.encoding, nil
	}

	h.tmp.Reset()

	encoding, err := l.header()
	h.tmp.Write(encoding)
	if err != nil {
		return nil, err
	}

	h.tmp.Write(nibblesToKeyLE(l.key))

	bytes, err := scale.Marshal(l.value)
	if err != nil {
		return nil, err
	}

	h.tmp.Write(bytes)
	l.encoding = h.tmp.Bytes()
	return h.tmp.Bytes(), nil
}
