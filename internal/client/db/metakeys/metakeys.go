package metakeys

// / Keys of entries in COLUMN_META.

// / Type of storage (full or light).
var Type = []byte("type")

// / Best block key.
var BestBlock = []byte("best")

// /// Last finalized block key.
var FinalizedBlock = []byte("final")

// /// Last finalized state key.
var FinalizedState = []byte("fstate")

// /// Block gap.
var BlockGap = []byte("gap")

// /// Genesis block hash.
var GenesisHash = []byte("gen")

// / Leaves prefix list key.
var LeafPrefix = []byte("leaf")

// / Children prefix list key.
var ChildrenPrefix = []byte("children")
