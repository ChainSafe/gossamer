package overlayedchanges

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
)

// / In memory array of storage values.
// pub type OffchainChangesCollection = Vec<((Vec<u8>, Vec<u8>), OffchainOverlayedChange)>;
type OffchainChangesCollection []struct {
	PrefixKey struct {
		Prefix []byte
		Key    []byte
	}
	ValueOperation offchain.OffchainOverlayedChange
}

// / Transaction index operation.
type IndexOperations interface {
	IndexOperationInsert | IndexOperationRenew
}

// / Transaction index operation.
type IndexOperation any

// / Insert transaction into index.
type IndexOperationInsert struct {
	/// Extrinsic index in the current block.
	Extrinsic uint32
	/// Data content hash.
	Hash []byte
	/// Indexed data size.
	Size uint32
}

// / Renew existing transaction storage.
type IndexOperationRenew struct {
	/// Extrinsic index in the current block.
	Extrinsic uint32
	/// Referenced index hash.
	Hash []byte
}
