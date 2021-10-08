package trie

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	MatchesLeaf = iota
	MatchesBranch
	NotFound
	IsChild
)

var (
	ErrDuplicateKeys         = errors.New("duplicate keys on verify proof")
	ErrIncompleteProof       = errors.New("incomplete proof")
	ErrNoMoreItemsOnIterable = errors.New("items iterable exhausted")
	ErrExhaustedNibbles      = errors.New("exhausted nibbles key")
	ErrExhaustedStack        = errors.New("no more itens to pop from stack")
	ErrValueMatchNotFound    = errors.New("value match not found")
	ErrExtraneousNode        = errors.New("the proof contains at least one extraneous node")
	ErrRootMismatch          = errors.New("computed root does not match with the given one")
)

type stackItem struct {
	value      []byte
	node       node
	rawNode    []byte
	path       []byte
	childIndex int
	children   [16]node
}

func newStackItem(path, raw []byte) (*stackItem, error) {
	decoded, err := decodeBytes(raw)
	if err != nil {
		return nil, err
	}

	return &stackItem{nil, decoded, raw, path, 0, [16]node{}}, nil
}

func (i *stackItem) advanceChildIndex(path []byte, proofI *proofIter) (*stackItem, error) {
	switch node := i.node.(type) {
	case *branch:
		if len(node.children) <= 0 {
			return nil, errors.New("branch node must to has children nodes")
		}

		if len(path) <= 0 {
			return nil, errors.New("descend node key is empty")
		}

		childIndex := (int)(path[len(path)-1])

		for i.childIndex < childIndex {
			child := node.children[i.childIndex]
			if child != nil {
				i.children[i.childIndex] = child
			}
			i.childIndex += 1
		}

		child := node.children[i.childIndex]
		return i.makeChildEntry(proofI, child, path)
	default:
		return nil, errors.New("node must be a branch node")
	}
}

func (i *stackItem) makeChildEntry(proofI *proofIter, child node, path []byte) (*stackItem, error) {
	if child == nil {
		return newStackItem(path, proofI.next())
	}
	encoded, err := encodeAndHash(child)
	if err != nil {
		return nil, err
	}

	return newStackItem(path, encoded)
}

func (i *stackItem) advanceItem(it *pairListIter) ([]byte, error) {
	for {
		item := it.peek()
		if item == nil {
			return nil, ErrNoMoreItemsOnIterable
		}

		nk := Nibbles(keyToNibbles(item.key))
		if bytes.HasPrefix(nk, i.path) {
			found, next, err := matchKeyToNode(nk, len(i.path), i.node)

			if err != nil {
				return nil, err
			} else if next != nil {
				return next, nil
			} else if found {
				i.value = item.value
			}

			it.next()
			continue
		}

		return nil, ErrNoMoreItemsOnIterable
	}
}

// matchKeyToNode return true if the leaf was found
// returns the byte array of the next node to keep searching
// returns error if the nibbles are exhausted or node key does not match
func matchKeyToNode(nk Nibbles, prefixOffset int, n node) (bool, []byte, error) {
	switch node := n.(type) {
	case nil:
		return false, nil, ErrValueMatchNotFound
	case *leaf:
		if nk.contains(node.key, uint(prefixOffset)) && len(nk) == prefixOffset+len(node.key) {
			return true, nil, nil
		}

		return false, nil, ErrValueMatchNotFound
	case *branch:
		if nk.contains(node.key, uint(prefixOffset)) {
			return matchKeyToBranchNode(nk, prefixOffset+len(node.key), node.children, node.value)
		}

		return false, nil, ErrValueMatchNotFound
	}

	return false, nil, ErrValueMatchNotFound
}

func matchKeyToBranchNode(nk Nibbles, prefixPlusKeyLen int, children [16]node, value []byte) (bool, []byte, error) {
	if len(nk) == prefixPlusKeyLen {
		return false, nil, nil
	}

	if len(nk) < prefixPlusKeyLen {
		return false, nil, ErrExhaustedNibbles
	}

	if children[nk[prefixPlusKeyLen]] == nil {
		return false, nil, ErrValueMatchNotFound
	}

	continueFrom := make([]byte, len(nk[prefixPlusKeyLen+1:]))
	copy(continueFrom, nk[prefixPlusKeyLen+1:])
	return false, continueFrom, nil
}

type stack []*stackItem

func (s *stack) push(si *stackItem) {
	*s = append(*s, si)
}

func (s *stack) pop() *stackItem {
	if len(*s) == 0 {
		return nil
	}

	i := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return i
}

type pair struct{ key, value []byte }
type PairList []*pair

func (pl *PairList) Add(k, v []byte) {
	*pl = append(*pl, &pair{k, v})
}

type pairListIter struct {
	idx int
	set []*pair
}

func (i *pairListIter) peek() *pair {
	if i.hasNext() {
		return i.set[i.idx]
	}

	return nil
}

func (i *pairListIter) next() *pair {
	if i.hasNext() {
		return i.set[i.idx]
		i.idx += 1
	}

	return nil
}

func (i *pairListIter) hasNext() bool {
	return len(i.set) < i.idx
}

func (pl *PairList) toIter() *pairListIter {
	return &pairListIter{0, *pl}
}

type proofIter struct {
	idx   int
	proof [][]byte
}

func (p *proofIter) next() []byte {
	if p.hasNext() {
		i := p.proof[p.idx]
		p.idx += 1
		return i
	}
	return nil
}

func (p *proofIter) hasNext() bool {
	return len(p.proof) < p.idx
}

func newProofIter(proof [][]byte) *proofIter {
	return &proofIter{0, proof}
}

func VerifyProof(root common.Hash, proof [][]byte, items PairList) (bool, error) {
	if len(proof) == 0 && len(items) == 0 {
		return true, nil
	}

	// check for duplicates
	for i := 1; i < len(items); i++ {
		if bytes.Equal(items[i].key, items[i-1].key) {
			return false, ErrDuplicateKeys
		}
	}

	proofI := newProofIter(proof)
	itemsI := items.toIter()

	var rootNode []byte
	if rootNode = proofI.next(); rootNode == nil {
		return false, ErrIncompleteProof
	}

	lastEntry, err := newStackItem([]byte{}, rootNode)
	if err != nil {
		return false, err
	}

	st := new(stack)

	for {
		descend, err := lastEntry.advanceItem(itemsI)

		if errors.Is(err, ErrNoMoreItemsOnIterable) {
			entry := st.pop()
			if entry == nil {
				if proofI.next() != nil {
					return false, ErrExtraneousNode
				}

				computedRoot := lastEntry.node.getHash()
				if !root.Equal(common.BytesToHash(computedRoot)) {
					return false, ErrRootMismatch
				}
				break
			}

			lastEntry = entry
			lastEntry.children[lastEntry.childIndex] = lastEntry.node
			lastEntry.childIndex += 1

		} else if err != nil {
			return false, err
		}

		nextEntry, err := lastEntry.advanceChildIndex(descend, proofI)
		if err != nil {
			return false, err
		}
		st.push(lastEntry)
		lastEntry = nextEntry
	}

	return false, nil
}
