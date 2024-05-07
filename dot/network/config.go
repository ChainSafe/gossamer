// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/lib/common"
)

// DefaultBasePath the default value for Config.BasePath
var DefaultBasePath = xdg.DataHome + "/gossamer"

const (
	// DefaultKeyFile the default value for KeyFile
	DefaultKeyFile = "node.key"

	// DefaultPort the default value for Config.Port
	DefaultPort = uint16(7000)

	// DefaultRandSeed the default value for Config.RandSeed (0 = non-deterministic)
	DefaultRandSeed = int64(0)

	// DefaultProtocolID the default value for Config.ProtocolID
	DefaultProtocolID = "/gossamer/gssmr/0"

	// DefaultRoles the default value for Config.Roles (0 = no network, 1 = full node)
	DefaultRoles = common.FullNodeRole

	// DefaultMinPeerCount is the default minimum peer count
	DefaultMinPeerCount = 5

	// DefaultMaxPeerCount is the default maximum peer count
	DefaultMaxPeerCount = 50

	// DefaultDiscoveryInterval is the default interval for searching for DHT peers
	DefaultDiscoveryInterval = time.Minute * 5

	defaultTxnBatchSize = 100
)

// DefaultBootnodes the default value for Config.Bootnodes
var DefaultBootnodes = []string(nil)

// Config is used to configure a network service
type Config struct {
	LogLvl  log.Level
	logger  Logger
	ErrChan chan<- error

	// BasePath the data directory for the node
	BasePath string
	// Roles a bitmap value that represents the different roles for the sender node (see Table D.2)
	Roles common.NetworkRole

	// Service interfaces
	BlockState         BlockState
	Syncer             Syncer
	TransactionHandler TransactionHandler

	// Used to specify the address broadcasted to other peers, and avoids using pubip.Get
	PublicIP string
	// Used to specify the dns broadcasted to other peers, and avoids using pubip.Get.
	// Only PublicIP or PublicDNS will be used
	PublicDNS string
	// Port the network port used for listening
	Port uint16
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
	// ListenAddress is the multiaddress to listen on
	ListenAddress string

	MinPeers int
	MaxPeers int

	DiscoveryInterval time.Duration

	// PersistentPeers is a list of multiaddrs which the node should remain connected to
	PersistentPeers []string

	// NodeKey is the private hex encoded Ed25519 key to build the p2p identity
	NodeKey string

	// privateKey the private key for the network p2p identity
	privateKey crypto.PrivKey

	// telemetryInterval how often to send telemetry metrics
	telemetryInterval time.Duration

	batchSize int // internal option

	// SlotDuration is the slot duration to produce a block
	SlotDuration time.Duration

	Telemetry Telemetry
	Metrics   metrics.IntervalConfig
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

	// set telemetryInterval to default
	if c.telemetryInterval.Microseconds() == 0 {
		c.telemetryInterval = time.Second * 5
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
	if c.NodeKey != "" {
		privateKeySeed, err := common.HexToBytes("0x" + c.NodeKey)
		if err != nil {
			return fmt.Errorf("parsing hex encoding of ed25519 private key: %w", err)
		}
		key := ed25519.NewKeyFromSeed(privateKeySeed)
		privateKey, err := crypto.UnmarshalEd25519PrivateKey(key)
		if err != nil {
			return fmt.Errorf("decoding ed25519 bytes: %w", err)
		}
		c.privateKey = privateKey
		return nil
	}

	if c.RandSeed == 0 {
		// attempt to load existing key
		key, err := loadKey(c.BasePath)
		if err != nil {
			return err
		}

		// generate key if no key exists
		if key == nil {
			c.logger.Infof(
				"Generating p2p identity with seed %d and key file %s",
				c.RandSeed, path.Join(c.BasePath, DefaultKeyFile))

			// generate key
			key, err = generateKey(c.RandSeed, c.BasePath)
			if err != nil {
				return err
			}
		}

		// set private key
		c.privateKey = key
	} else {
		c.logger.Infof(
			"Generating p2p identity with seed %d and key file %s",
			c.RandSeed, path.Join(c.BasePath, DefaultKeyFile))

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
		c.logger.Warn("ProtocolID not defined, using default protocol id " + DefaultProtocolID)
		c.ProtocolID = DefaultProtocolID
	}

	// append "/" to front of protocol ID, if not already there
	if c.ProtocolID[:1] != "/" {
		c.ProtocolID = "/" + c.ProtocolID
	}

	return nil
}
