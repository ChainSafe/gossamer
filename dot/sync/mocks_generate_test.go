// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . BlockState,StorageState,CodeSubstitutedState,TransactionState,BabeVerifier,FinalityGadget,BlockImportHandler,Network,RuntimeInstance
//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client
//go:generate mockgen -destination=mock_chain_processor_test.go -package=$GOPACKAGE . ChainProcessor
//go:generate mockgen -destination=mock_chain_sync_test.go -package $GOPACKAGE -source chain_sync.go . ChainSync,workHandler
//go:generate mockgen -destination=mock_disjoint_block_set_test.go -package=$GOPACKAGE . DisjointBlockSet
