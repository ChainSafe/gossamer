// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . Table,ImplicitView
//go:generate mockgen -destination=mocks_blockstate_test.go -package=$GOPACKAGE . Blockstate
