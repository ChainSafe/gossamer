// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package runtime

// PageSize is 65kb
const PageSize = 65536

//go:generate mockery --name Memory --structname mockMemory --case underscore --inpackage --filename mock_memory_test.go

// Memory is a raw memory interface
type Memory interface {
	Data() []byte
	Length() uint32
	Grow(uint32) error
}
