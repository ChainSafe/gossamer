// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

//go:generate mockgen -destination=mocks_test.go -package $GOPACKAGE . BlockState,GrandpaState,Network
//go:generate mockgen -source=finalisation.go -destination=mock_ephemeral_service_test.go -package $GOPACKAGE . ephemeralService
//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mocks_runtime_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
