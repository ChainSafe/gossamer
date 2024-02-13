// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . Network
//go:generate mockgen -destination=overseer_mocks_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/dot/parachain/overseer OverseerI
//go:generate mockgen -destination=mock_blockstate_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/dot/parachain/overseer BlockState
