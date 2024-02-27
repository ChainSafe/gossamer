package trietypes

import triedb "github.com/ChainSafe/gossamer/internal/trie-db"

type TrieDBBuilder[Hash comparable] triedb.TrieDBBuilder[Hash]

func (tdbb TrieDBBuilder[Hash]) Build() triedb.Trie[Hash] {
	return triedb.TrieDBBuilder[Hash](tdbb).Build()
}
