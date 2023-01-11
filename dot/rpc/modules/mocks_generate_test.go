// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . StorageAPI,BlockAPI
//go:generate mockgen -source=interfaces_mock_source.go -destination=mocks_local_test.go -package=$GOPACKAGE
//go:generate mockgen -destination=mocks/mocks.go -package mocks . StorageAPI,BlockAPI,NetworkAPI,BlockProducerAPI,TransactionStateAPI,CoreAPI,SystemAPI,BlockFinalityAPI,RuntimeStorageAPI,SyncStateAPI
//go:generate mockgen -destination=mock_sync_api_test.go -package $GOPACKAGE . SyncAPI
//go:generate mockgen -destination=mocks_babe_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/babe BlockImportHandler
