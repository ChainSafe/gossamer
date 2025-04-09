// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

//go:generate mockgen -destination=mocks_network_test.go -package=$GOPACKAGE . Network
//go:generate mockgen -destination=mocks_blockstate_test.go -package=$GOPACKAGE . BlockState
//go:generate mockgen -destination=mocks_session_cache_test.go -package=$GOPACKAGE . SessionCache
//go:generate mockgen -destination=mocks_runtime_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
//go:generate mockgen -destination=mocks_keystore_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/keystore Keystore
