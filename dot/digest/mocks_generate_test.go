// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

//go:generate mockgen -source=interfaces_mock_source.go -destination=mocks_local_test.go -package=$GOPACKAGE
//go:generate mockgen -destination=mock_grandpa_test.go -package $GOPACKAGE . GrandpaState
//go:generate mockgen -destination=mock_epoch_state_test.go -package $GOPACKAGE . EpochState
