// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . StorageAPI,BlockAPI,Telemetry
//go:generate mockgen -destination=mocks/mocks.go -package mocks . StorageAPI,BlockAPI,NetworkAPI,BlockProducerAPI,TransactionStateAPI,CoreAPI,SystemAPI,BlockFinalityAPI,RuntimeStorageAPI,SyncStateAPI
//go:generate mockgen -destination=mock_sync_api_test.go -package $GOPACKAGE . SyncAPI
//go:generate mockgen -destination=mock_syncer_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/network Syncer
//go:generate mockgen -destination=mocks_babe_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/babe BlockImportHandler
