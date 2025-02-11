// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package warpsync

//go:generate mockgen -destination=mocks_test.go -package $GOPACKAGE . GrandpaState
//go:generate mockgen -destination=mocks_block_state_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/state BlockState
