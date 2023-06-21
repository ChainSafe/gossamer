// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import "github.com/ChainSafe/gossamer/pkg/scale"

type ApprovalDistribution ApprovalDistributionMessage

func (ad ApprovalDistribution) Index() uint {
	return 4
}

type ValidationProtocol scale.VaryingDataType

func NewValidationProtocolVDT() ValidationProtocol {
	vdt, err := scale.NewVaryingDataType(ApprovalDistribution{}, Approvals{})
	if err != nil {
		panic(err)
	}
	return ValidationProtocol(vdt)
}

// Value returns the value from the underlying VaryingDataType
func (vp *ValidationProtocol) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*vp)
	return vdt.Value()
}
