// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

//go:generate mockery --srcpkg=github.com/ChainSafe/gossamer/internal/client/api --name=Backend --case=snake --with-expecter=true
//go:generate mockery --srcpkg=github.com/ChainSafe/gossamer/internal/primitives/blockchain --name=Backend --case=snake --structname=BlockchainBackend --filename=blockchain_backend.go --with-expecter=true
