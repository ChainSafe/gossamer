// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mock_grandpa_test.go -package $GOPACKAGE . GrandpaState
