// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

//go:generate mockgen -destination=mocks_test.go -package $GOPACKAGE . Telemetry,BlockStateDatabase,Observer
//go:generate mockgen -destination=mocks_runtime_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
//go:generate mockgen -destination=mock_gauge_test.go -package $GOPACKAGE github.com/prometheus/client_golang/prometheus Gauge
//go:generate mockgen -destination=mock_counter_test.go -package $GOPACKAGE github.com/prometheus/client_golang/prometheus Counter
//go:generate mockgen -destination=db_getter_mocks_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/trie DBGetter
