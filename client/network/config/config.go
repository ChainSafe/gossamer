package config

// Role of the local node.
type Role uint

const (
	/// Regular full node.
	Full Role = iota
	/// Actual authority.
	Authority
)
