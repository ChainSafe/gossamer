package trie

const TrieValueNodeThreshold uint32 = 32

type TrieLayout interface {
	maxInlineValue() *uint32
}

type TrieLayoutV0 struct{}

func (tl TrieLayoutV0) maxInlineValue() *uint32 {
	return nil
}

type TrieLayoutV1 struct{}

func (tl TrieLayoutV1) maxInlineValue() *uint32 {
	value := uint32(TrieValueNodeThreshold)
	return &value
}
