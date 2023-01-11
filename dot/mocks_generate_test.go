// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

//go:generate mockgen -destination=mock_block_state_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/network BlockState
//go:generate mockgen -source=interfaces_mock_source.go -destination=mocks_local_test.go -package=$GOPACKAGE
