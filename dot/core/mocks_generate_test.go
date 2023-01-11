// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

//go:generate mockgen -destination=mocks_test.go -package $GOPACKAGE . BlockState,StorageState,TransactionState,Network,CodeSubstitutedState,RuntimeInstance
//go:generate mockgen -source=interfaces_mock_source.go -destination=mocks_local_test.go -package $GOPACKAGE
