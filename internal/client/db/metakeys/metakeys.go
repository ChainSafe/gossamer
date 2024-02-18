package metakeys

// / Keys of entries in COLUMN_META.
//
//	pub mod meta_keys {
//		/// Type of storage (full or light).
//		pub const TYPE: &[u8; 4] = b"type";
//
// / Best block key.
var BestBlock = []byte("best")

// /// Last finalized block key.
// pub const FINALIZED_BLOCK: &[u8; 5] = b"final";
var FinalizedBlock = []byte("final")

// /// Last finalized state key.
// pub const FINALIZED_STATE: &[u8; 6] = b"fstate";
var FinalizedState = []byte("fstate")

// /// Block gap.
// pub const BLOCK_GAP: &[u8; 3] = b"gap";
var BlockGap = []byte("gap")

// /// Genesis block hash.
// pub const GENESIS_HASH: &[u8; 3] = b"gen";
var GenesisHash = []byte("gen")

// / Leaves prefix list key.
var LeafPrefix = []byte("leaf")

// 	/// Children prefix list key.
// 	pub const CHILDREN_PREFIX: &[u8; 8] = b"children";
// }
