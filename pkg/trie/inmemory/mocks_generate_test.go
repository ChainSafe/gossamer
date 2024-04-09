// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

//go:generate mockgen -destination=db_getter_mocks_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/pkg/trie/db DBGetter
