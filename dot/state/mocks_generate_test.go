// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

//go:generate mockgen -destination=mocks_test.go -package $GOPACKAGE . Telemetry,Observer
//go:generate mockgen -source=interfaces_mock_source.go -destination=mocks_local_test.go -package $GOPACKAGE
//go:generate mockgen -destination=mock_gauge_test.go -package $GOPACKAGE github.com/prometheus/client_golang/prometheus Gauge
//go:generate mockgen -destination=mock_counter_test.go -package $GOPACKAGE github.com/prometheus/client_golang/prometheus Counter
