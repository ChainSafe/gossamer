// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scraping

//go:generate mockgen -destination=mock_runtime_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/parachain/runtime RuntimeInstance