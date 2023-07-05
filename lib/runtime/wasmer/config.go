// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// Config is the configuration used to create a Wasmer runtime instance.
type Config struct {
	Storage     runtime.Storage
	Keystore    *keystore.GlobalKeystore
	LogLvl      log.Level
	Role        common.NetworkRole
	NodeStorage runtime.NodeStorage
	Network     runtime.BasicNetwork
	Transaction runtime.TransactionState
	CodeHash    common.Hash
	testVersion *runtime.Version
}

// SetTestVersion sets the test version for the runtime.
// WARNING: This should only be used for testing purposes.
// The *testing.T argument is only required to enforce this function
// to be used in tests only.
func (c *Config) SetTestVersion(t *testing.T, version runtime.Version) {
	if t == nil {
		panic("*testing.T argument cannot be nil. Please don't use this function outside of Go tests.")
	}
	c.testVersion = &version
}
