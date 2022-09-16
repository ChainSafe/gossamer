// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

//go:generate mockgen -destination=mocks.go -package $GOPACKAGE . BlockState,StorageState,TransactionState,Network,EpochState,CodeSubstitutedState,RuntimeInstance
