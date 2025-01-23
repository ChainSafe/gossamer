// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . Network
//go:generate mockgen -destination=mock_request_maker_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/dot/network RequestMaker
