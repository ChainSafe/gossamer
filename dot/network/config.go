// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"errors"
	"path"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/crypto"
)

const (
	// DefaultKeyFile the default value for KeyFile
	DefaultKeyFile = "node.key"

	// DefaultBasePath the default value for Config.BasePath
	DefaultBasePath = "~/.gossamer/gssmr"

	// DefaultPort the default value for Config.Port
	DefaultPort = uint32(7000)

	// DefaultRandSeed the default value for Config.RandSeed (0 = non-deterministic)
	DefaultRandSeed = int64(0)

	// DefaultProtocolID the default value for Config.ProtocolID
	DefaultProtocolID = "/gossamer/gssmr/0"

	// DefaultRoles the default value for Config.Roles (0 = no network, 1 = full node)
	DefaultRoles = byte(1)

	// DefaultMinPeerCount is the default minimum peer count
	DefaultMinPeerCount = 5

	// DefaultMaxPeerCount is the default maximum peer count
	DefaultMaxPeerCount = 50
)

// DefaultBootnodes the default value for Config.Bootnodes
var DefaultBootnodes = []string(nil)

// Config is used to configure a network service
type Config struct {
	LogLvl  log.Lvl
	logger  log.Logger
	ErrChan chan<- error

	// BasePath the data directory for the node
	BasePath string
	// Roles a bitmap value that represents the different roles for the sender node (see Table D.2)
	Roles byte

	// Service interfaces
	BlockState         BlockState
	Syncer             Syncer
	TransactionHandler TransactionHandler

	// Port the network port used for listening
	Port uint32
	// RandSeed the seed used to generate the network p2p identity (0 = non-deterministic random seed)
	RandSeed int64
	// Bootnodes the peer addresses used for bootstrapping
	Bootnodes []string
	// ProtocolID the protocol ID for network messages
	ProtocolID string
	// NoBootstrap disables bootstrapping
	NoBootstrap bool
	// NoMDNS disables MDNS discovery
	NoMDNS bool

	MinPeers int
	MaxPeers int

	// PersistentPeers is a list of multiaddrs which the node should remain connected to
	PersistentPeers []string

	// privateKey the private key for the network p2p identity
	privateKey crypto.PrivKey

	// PublishMetrics enables collection of network metrics
	PublishMetrics bool
}

// build checks the configuration, sets up the private key for the network service,
// and applies default values where appropriate
func (c *Config) build() error {

	// check state configuration
	err := c.checkState()
	if err != nil {
		return err
	}

	if c.BasePath == "" {
		c.BasePath = DefaultBasePath
	}

	if c.Roles == 0 {
		c.Roles = DefaultRoles
	}

	if c.Port == 0 {
		c.Port = DefaultPort
	}

	// build identity configuration
	err = c.buildIdentity()
	if err != nil {
		return err
	}

	// build protocol configuration
	err = c.buildProtocol()
	if err != nil {
		return err
	}

	// check bootnoode configuration
	if !c.NoBootstrap && len(c.Bootnodes) == 0 {
		c.logger.Warn("Bootstrap is enabled but no bootstrap nodes are defined")
	}

	return nil
}

func (c *Config) checkState() (err error) {
	// set NoStatus to true if we don't need BlockState
	if c.BlockState == nil {
		err = errors.New("failed to build configuration: BlockState required")
	}

	return err
}

// buildIdentity attempts to load the private key required to start the network
// service, if a key does not exist or cannot be loaded, it creates a new key
// using the random seed (if random seed is not set, creates new random key)
func (c *Config) buildIdentity() error {
	if c.RandSeed == 0 {

		// attempt to load existing key
		key, err := loadKey(c.BasePath)
		if err != nil {
			return err
		}

		// generate key if no key exists
		if key == nil {
			c.logger.Info(
				"Generating p2p identity",
				"RandSeed", c.RandSeed,
				"KeyFile", path.Join(c.BasePath, DefaultKeyFile),
			)

			// generate key
			key, err = generateKey(c.RandSeed, c.BasePath)
			if err != nil {
				return err
			}
		}

		// set private key
		c.privateKey = key
	} else {
		c.logger.Info(
			"Generating p2p identity from seed",
			"RandSeed", c.RandSeed,
			"KeyFile", path.Join(c.BasePath, DefaultKeyFile),
		)

		// generate temporary deterministic key
		key, err := generateKey(c.RandSeed, c.BasePath)
		if err != nil {
			return err
		}

		// set private key
		c.privateKey = key
	}

	return nil
}

// buildProtocol verifies and applies defaults to the protocol configuration
func (c *Config) buildProtocol() error {
	if c.ProtocolID == "" {
		c.logger.Warn(
			"ProtocolID not defined, using DefaultProtocolID",
			"DefaultProtocolID", DefaultProtocolID,
		)
		c.ProtocolID = DefaultProtocolID
	}

	// append "/" to front of protocol ID, if not already there
	if c.ProtocolID[:1] != "/" {
		c.ProtocolID = "/" + c.ProtocolID
	}

	return nil
}
