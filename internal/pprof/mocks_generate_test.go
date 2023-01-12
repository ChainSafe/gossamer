// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

//go:generate mockgen -destination=logger_mock_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/internal/httpserver Logger
//go:generate mockgen -source=interfaces_mock_source.go -destination=mocks_local_test.go -package $GOPACKAGE
