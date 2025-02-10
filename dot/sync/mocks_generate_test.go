// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . Telemetry,StorageState,TransactionState,BabeVerifier,FinalityGadget,BlockImportHandler,Network
//go:generate mockgen -destination=mock_request_maker.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/network RequestMaker
//go:generate mockgen -destination=mock_importer.go -source=fullsync.go -package=sync
//go:generate mockgen -destination=mock_block_state_maker.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/state BlockState
