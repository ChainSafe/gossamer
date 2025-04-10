// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package chainspec

//go:generate mockgen -destination=mock_executor_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/internal/client/executor RuntimeVersionOf
