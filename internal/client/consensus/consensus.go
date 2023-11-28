package consensus

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
