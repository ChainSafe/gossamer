// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . StorageAPI,BlockAPI
//go:generate mockgen -destination=mock_code_substituted_state_test.go -package modules github.com/ChainSafe/gossamer/dot/core CodeSubstitutedState
//go:generate mockgen -destination=mock_block_state_test.go -package modules github.com/ChainSafe/gossamer/dot/network BlockState
//go:generate mockgen -destination=mock_syncer_test.go -package modules github.com/ChainSafe/gossamer/dot/network Syncer
//go:generate mockgen -destination=mock_transaction_handler_test.go -package modules github.com/ChainSafe/gossamer/dot/network TransactionHandler
//go:generate mockgen -destination=mock_network_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/core Network
