// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . BlockState,StorageState,TransactionState,BabeVerifier,FinalityGadget,BlockImportHandler,Network
//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mock_runtime_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
//go:generate mockgen -destination=mock_chain_sync_test.go -package $GOPACKAGE -source chain_sync.go . ChainSync
//go:generate mockgen -destination=mock_disjoint_block_set_test.go -package=$GOPACKAGE . DisjointBlockSet
//go:generate mockgen -destination=mock_request.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/network RequestMaker
