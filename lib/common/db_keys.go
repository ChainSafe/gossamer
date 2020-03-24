package common

var (
	// BestBlockHashKey is the db location the hash of the best block header.
	BestBlockHashKey = []byte("best_hash")
	// LatestStorageHashKey is the db location of the hash of the latest storage trie.
	LatestStorageHashKey = []byte("latest_storage_hash")
	// GenesisDataKey is the db location of the genesis data.
	GenesisDataKey = []byte("genesis_data")
	// BlockTreeKey is the db location of the encoded block tree structure.
	BlockTreeKey = []byte("block_tree")
)
