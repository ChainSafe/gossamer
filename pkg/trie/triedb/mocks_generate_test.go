package triedb

//go:generate mockgen -destination=mock_trie_cache.go -package $GOPACKAGE . TrieCache[hash.H256]
