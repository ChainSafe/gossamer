// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . ServiceRegisterer
//go:generate mockgen -destination=mock_block_state_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/network BlockState
//go:generate mockgen -source=node.go -destination=mock_node_builder_test.go -package=$GOPACKAGE
//go:generate mockgen -destination=mock_service_builder_test.go -package $GOPACKAGE . ServiceBuilder
