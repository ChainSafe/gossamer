package consensus

// / Block status.
type BlockStatus uint

const (
	/// Added to the import queue.
	BlockStatusQueued BlockStatus = iota
	/// Already in the blockchain and the state is available.
	BlockStatusInChainWithState
	/// In the blockchain, but the state is not available.
	BlockStatusInChainPruned
	/// Block or parent is known to be bad.
	BlockStatusKnownBad
	/// Not in the queue or the blockchain.
	BlockStatusUnknown
)

// / Block data origin.
type BlockOrigin uint

const (
	/// Genesis block built into the client.
	BlockOriginGenesis BlockOrigin = iota
	/// Block is part of the initial sync with the network.
	BlockOriginNetworkInitialSync
	/// Block was broadcasted on the network.
	BlockOriginNetworkBroadcast
	/// Block that was received from the network and validated in the consensus process.
	BlockOriginConsensusBroadcast
	/// Block that was collated by this node.
	BlockOriginOwn
	/// Block was imported from a file.
	BlockOriginFile
)
