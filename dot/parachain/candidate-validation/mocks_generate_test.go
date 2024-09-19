// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . PoVRequestor
//go:generate mockgen -destination=mocks_blockstate_test.go -package=$GOPACKAGE . BlockState
//go:generate mockgen -destination=mocks_instance_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
