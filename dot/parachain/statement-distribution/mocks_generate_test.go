// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

//go:generate mockgen -destination=mocks_implicitview_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/dot/parachain/util ImplicitView
//go:generate mockgen -destination=mocks_statement_store_test.go -package=$GOPACKAGE . statementStore
