// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

var TrieValueNodeThreshold uint32 = 32

type TrieLayout interface {
	MaxInlineValue() *uint32
}

type TrieLayoutV0 struct{}

func (tl TrieLayoutV0) MaxInlineValue() *uint32 {
	return nil
}

type TrieLayoutV1 struct{}

func (tl TrieLayoutV1) MaxInlineValue() *uint32 {
	return &TrieValueNodeThreshold
}
