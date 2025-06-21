// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package adapter

//go:generate mockery --name=Client --case=snake --with-expecter=true
//go:generate mockery --name=Backend --case=snake --with-expecter=true
//go:generate mockery --name=ClientAdapterDB --case=snake --with-expecter=true
//go:generate mockery --srcpkg=github.com/ChainSafe/gossamer/internal/primitives/state-machine --name=Backend --case=snake --structname=StatemachineBackend --filename=statemachine_backend.go --with-expecter=true
