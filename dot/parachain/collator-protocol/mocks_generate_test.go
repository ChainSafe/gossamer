// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

<<<<<<<< HEAD:dot/parachain/collator-protocol/mocks_generate_test.go
//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . Network
========
//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . PoVRequestor
//go:generate mockgen -destination=mocks_runtime_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/parachain/runtime RuntimeInstance
>>>>>>>> 7c50d399 (moved `lib/parachain` to `dot/parachain` (#3429)):dot/parachain/mocks_generate_test.go
