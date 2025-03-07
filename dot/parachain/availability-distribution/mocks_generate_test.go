// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

//go:generate mockgen -destination=mocks_network_test.go -package=$GOPACKAGE . Network
//go:generate mockgen -destination=mocks_blockstate_test.go -package=$GOPACKAGE . BlockState
