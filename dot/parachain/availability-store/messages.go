// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// AvailabilityStoreMessage represents the possible availability store subsystem message
type AvailabilityStoreMessage scale.VaryingDataType

// QueryAvailableData query a AvailableData from the AV store
type QueryAvailableData struct {
	CandidateHash common.Hash
	AvailableData AvailableData
}

// Index returns the index of varying data type
func (QueryAvailableData) Index() uint {
	return 0
}

// NewCollationFetchingResponse returns a new collation fetching response varying data type
func NewAvailabilityStoreMessage() AvailabilityStoreMessage {
	vdt := scale.MustNewVaryingDataType(QueryAvailableData{})
	return AvailabilityStoreMessage(vdt)
}

type AvailableData struct{} // Define your AvailableData type
