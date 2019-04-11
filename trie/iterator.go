package trie

import (
	"fmt"
)

type Iterator struct {
	current node
}

func (t *Trie) Print() {
	t.print(t.root)
}

func (t *Trie) print(current node) {
	switch c := current.(type) {
	case *branch:
		fmt.Printf("branch pk %x children %b value %s\n", c.key, c.childrenBitmap(), c.value)
		for _, child := range c.children {
			// if child != nil {
			// 	fmt.Printf("child at %x\n", i)
			// }
			t.print(child)
		}
	case *leaf:
		fmt.Printf("leaf pk %x val %s\n", c.key, c.value)
	default:
		// do nothing
	}
}