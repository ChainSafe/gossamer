// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . Network
//go:generate mockgen -destination=overseer_mocks_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/dot/parachain/overseer OverseerI
//go:generate mockgen -destination=mock_blockstate_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/dot/parachain/overseer BlockState
//go:generate mockgen -destination=mock-network/mock_block_state.go -package mock_network github.com/ChainSafe/gossamer/dot/network BlockState
//go:generate mockgen -destination=mock-network/mock_syncer.go -package mock_network github.com/ChainSafe/gossamer/dot/network Syncer
//go:generate mockgen -destination=mock-network/mock_transaction_handler.go -package mock_network github.com/ChainSafe/gossamer/dot/network TransactionHandler
//go:generate mockgen -destination=mock-network/mock_telemetry.go -package mock_network github.com/ChainSafe/gossamer/dot/network Telemetry
