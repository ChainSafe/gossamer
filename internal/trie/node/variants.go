// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

type variant struct {
	bits byte
	mask byte
}

// Node variants
// See https://spec.polkadot.network/#defn-node-header
var (
	leafVariant = variant{ // leaf 01
		bits: 0b0100_0000,
		mask: 0b1100_0000,
	}
	branchVariant = variant{ // branch 10
		bits: 0b1000_0000,
		mask: 0b1100_0000,
	}
	branchWithValueVariant = variant{ // branch 11
		bits: 0b1100_0000,
		mask: 0b1100_0000,
	}
	leafWithHashedValueVariant = variant{ // leaf containing hashes 001
		bits: 0b0010_0000,
		mask: 0b1110_0000,
	}
	branchWithHashedValueVariant = variant{ // branch containing hashes 0001
		bits: 0b0001_0000,
		mask: 0b1111_0000,
	}
	emptyVariant = variant{ // empty 0000 0000
		bits: 0b0000_0000,
		mask: 0b1111_1111,
	}
	compactEncodingVariant = variant{ // compact encoding 0001 0000
		bits: 0b0000_0001,
		mask: 0b1111_1111,
	}
)

// partialKeyLengthHeaderMask returns the partial key length
// header bit mask corresponding to the variant header bit mask.
// For example for the leaf variant with variant mask 1100_0000,
// the partial key length header mask returned is 0011_1111.
func (v variant) partialKeyLengthHeaderMask() byte {
	return ^v.mask
}

func (v variant) String() string {
	switch v {
	case leafVariant:
		return "Leaf"
	case leafWithHashedValueVariant:
		return "LeafWithHashedValue"
	case branchVariant:
		return "Branch"
	case branchWithValueVariant:
		return "BranchWithValue"
	case branchWithHashedValueVariant:
		return "BranchWithHashedValue"
	case emptyVariant:
		return "Empty"
	case compactEncodingVariant:
		return "Compact"
	default:
		return "Not reachable"
	}

}
