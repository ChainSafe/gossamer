package triedb

import "fmt"

type HashOut interface {
	comparable
	ToBytes() []byte
}

type NodeCodec[H HashOut] interface {
	HashedNullNode() H
	EmptyNode() []byte
	LeafNode(partialKey []byte, numberNibble uint, value Value[H]) []byte
	BranchNodeNibbled(partialKey []byte, numberNibble uint, children [16]NodeHandle[H], value Value[H]) []byte
}

func EncodeNode[H HashOut](node Node[H], codec NodeCodec[H]) []byte {
	switch n := node.(type) {
	case Empty:
		return codec.EmptyNode()
	case Leaf[H]:
		return codec.LeafNode(n.partialKey.RightIter(), n.partialKey.Len(), n.value)
	case NibbledBranch[H]:
		return codec.BranchNodeNibbled(n.partialKey.RightIter(), n.partialKey.Len(), n.childs, n.value)
	default:
		panic(fmt.Sprintf("unknown node type %s", n.Type()))
	}
}
