// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . Telemetry,BlockState,StorageState,TransactionState,BabeVerifier,FinalityGadget,BlockImportHandler,Network,Importer
//go:generate mockgen -destination=mock_request_maker.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/network RequestMaker
