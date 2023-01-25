// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

//go:generate mockgen -destination=logger_mock_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/internal/httpserver Logger
//go:generate mockgen -destination=runner_mock_test.go -package $GOPACKAGE . Runner
