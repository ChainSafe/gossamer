// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

//go:generate mockgen -destination=mocks/mocks.go -package mocks . Instance,TransactionState
//go:generate mockgen -destination=mocks_test.go -package $GOPACKAGE . Memory
