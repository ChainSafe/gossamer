// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

//go:generate mockgen -package=$GOPACKAGE -destination=mocks_test.go . JournalDatabase,Getter,PutDeleter,ChainDBNewBatcher,Logger,BlockState
//go:generate mockgen -package=$GOPACKAGE -destination=mocks_chaindb_test.go github.com/ChainSafe/chaindb Batch
