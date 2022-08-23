// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

//go:generate mockgen -destination=mocks_runtime_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
//go:generate mockgen -destination=mock_test.go -package $GOPACKAGE . BlockState,StorageState,TransactionState,Network,EpochState,CodeSubstitutedState
