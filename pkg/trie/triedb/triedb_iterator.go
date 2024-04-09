package triedb

import (
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

type IteratorState struct {
	node     codec.Node // actual node
	childIdx int        // index of the child node
}

type TrieDBIterator struct {
	db         *TrieDB // trie to iterate over
	currentKey common.Hash
	nodeStack  []IteratorState // Pending nodes to visit
	finished   bool
}

func NewTrieDBIterator(trie *TrieDB) *TrieDBIterator {
	return &TrieDBIterator{
		db:         trie,
		currentKey: trie.rootHash,
		nodeStack:  make([]IteratorState, 0),
	}
}

func (i *TrieDBIterator) HasNext() bool {
	return !i.finished
}

func (i *TrieDBIterator) Next() (codec.Node, error) {
	if i.finished {
		return nil, errors.New("iterator has finished")
	}

	//Initial case
	if len(i.nodeStack) == 0 {
		rootNode, err := i.db.getRootNode()
		if err != nil {
			return nil, err
		}
		i.nodeStack = append(i.nodeStack, IteratorState{node: rootNode, childIdx: -1})
	}

	for len(i.nodeStack) > 0 {
		currentState := i.nodeStack[len(i.nodeStack)-1]
		currentNode := currentState.node

		switch n := currentNode.(type) {
		case codec.Leaf:
			i.nodeStack = i.nodeStack[:len(i.nodeStack)-1]
			return n, nil
		case codec.Branch:
			if currentState.childIdx+1 < len(n.Children) {
				currentState.childIdx++
				childNode, err := i.db.getNode(n.Children[currentState.childIdx], n.PartialKey)
				if err != nil {
					return nil, err
				}
				i.nodeStack = append(i.nodeStack, IteratorState{node: childNode, childIdx: -1})
				continue
			}
		}
		i.nodeStack = i.nodeStack[:len(i.nodeStack)-1]
	}

	i.finished = true
	return nil, errors.New("no more elements")
}
