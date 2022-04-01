package node

import "io"

// Option is an option to use with the `New` constructor.
type Option func(node *Node)

// SetConfig sets the config path for the node.
func SetConfig(configPath string) Option {
	return func(node *Node) {
		node.configPath = configPath
	}
}

// SetGenesis sets the genesis path for the node.
func SetGenesis(genesisPath string) Option {
	return func(node *Node) {
		node.genesisPath = genesisPath
	}
}

// SetBasePath sets the base path for the node.
func SetBasePath(basePath string) Option {
	return func(node *Node) {
		node.basePath = basePath
	}
}

// SetIndex sets the index for the node.
func SetIndex(index int) Option {
	return func(node *Node) {
		node.index = intPtr(index)
	}
}

// SetBabeLead sets the babe lead boolean for the node.
func SetBabeLead(babeLead bool) Option {
	return func(node *Node) {
		node.babeLead = boolPtr(babeLead)
	}
}

// SetWebsocket sets the websocket boolean for the node.
func SetWebsocket(websocket bool) Option {
	return func(node *Node) {
		node.websocket = boolPtr(websocket)
	}
}

// SetWriter sets the writer for the node.
func SetWriter(writer io.Writer) Option {
	return func(node *Node) {
		node.writer = writer
	}
}
