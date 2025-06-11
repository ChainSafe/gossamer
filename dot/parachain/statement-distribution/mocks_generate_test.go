// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

//go:generate mockgen -destination=mocks_implicitview_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/dot/parachain/util ImplicitView
//go:generate mockgen -destination=mocks_statement_store_test.go -package=$GOPACKAGE . statementStore
//go:generate mockgen -destination=mocks_candidates_tracker_test.go -package=$GOPACKAGE . candidatesTracker
//go:generate mockgen -destination=mocks_cluster_tracker_test.go -package=$GOPACKAGE . clusterTracker
//go:generate mockgen -destination=mocks_req_manager_test.go -package=$GOPACKAGE . requestManager
//go:generate mockgen -destination=mocks_block_state_test.go -package=$GOPACKAGE . blockState
//go:generate mockgen -destination=mocks_instance_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
