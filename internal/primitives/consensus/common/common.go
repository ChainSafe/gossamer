package common

type BlockOrigin uint

const (
	NetworkInitialSync BlockOrigin = iota
	NetworkBroadcast
)
