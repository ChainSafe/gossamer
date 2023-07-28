// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

type Variant struct {
	bits byte
	mask byte
}

// Node variants
// See https://spec.polkadot.network/#defn-node-header
var (
	LeafVariant = Variant{ // leaf 01
		bits: 0b0100_0000,
		mask: 0b1100_0000,
	}
	BranchVariant = Variant{ // branch 10
		bits: 0b1000_0000,
		mask: 0b1100_0000,
	}
	BranchWithValueVariant = Variant{ // branch 11
		bits: 0b1100_0000,
		mask: 0b1100_0000,
	}
	LeafWithHashedValueVariant = Variant{ // leaf containing hashes 001
		bits: 0b0010_0000,
		mask: 0b1110_0000,
	}
	BranchWithHashedValueVariant = Variant{ // branch containing hashes 0001
		bits: 0b0001_0000,
		mask: 0b1111_0000,
	}
	EmptyVariant = Variant{ // empty 0000 0000
		bits: 0b0000_0000,
		mask: 0b1111_1111,
	}
	compactEncodingVariant = Variant{ // compact encoding 0001 0000
		bits: 0b0000_0001,
		mask: 0b1111_1111,
	}
	invalidVariant = Variant{
		bits: 0b0000_0000,
		mask: 0b0000_0000,
	}
)

// partialKeyLengthHeaderMask returns the partial key length
// header bit mask corresponding to the variant header bit mask.
// For example for the leaf variant with variant mask 1100_0000,
// the partial key length header mask returned is 0011_1111.
func (v Variant) partialKeyLengthHeaderMask() byte {
	return ^v.mask
}

func (v Variant) String() string {
	switch v {
	case LeafVariant:
		return "Leaf"
	case LeafWithHashedValueVariant:
		return "LeafWithHashedValue"
	case BranchVariant:
		return "Branch"
	case BranchWithValueVariant:
		return "BranchWithValue"
	case BranchWithHashedValueVariant:
		return "BranchWithHashedValue"
	case EmptyVariant:
		return "Empty"
	case compactEncodingVariant:
		return "Compact"
	case invalidVariant:
		return "Invalid"
	default:
		return "Not reachable"
	}

}
