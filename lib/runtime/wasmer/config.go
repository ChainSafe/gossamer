// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// Config is the configuration used to create a
// Wasmer runtime instance.
type Config struct {
	Storage     runtime.Storage
	Keystore    *keystore.GlobalKeystore
	LogLvl      log.Level
	Role        common.Roles
	NodeStorage runtime.NodeStorage
	Network     runtime.BasicNetwork
	Transaction runtime.TransactionState
	CodeHash    common.Hash
	testVersion *runtime.Version
}
