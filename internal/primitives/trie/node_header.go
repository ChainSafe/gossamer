package trie

import "github.com/ChainSafe/gossamer/internal/saturating"

// / NodeHeader without content
type nodeKind uint

const (
	nodeKindLeaf nodeKind = iota
	nodeKindBranchNoValue
	nodeKindBranchWithValue
	nodeKindHashedValueLeaf
	nodeKindHashedValueBranch
)

// / Returns an iterator over encoded bytes for node header and size.
// / Size encoding allows unlimited, length inefficient, representation, but
// / is bounded to 16 bit maximum value to avoid possible DOS.
// pub(crate) fn size_and_prefix_iterator(
//
//	size: usize,
//	prefix: u8,
//	prefix_mask: usize,
//
// ) -> impl Iterator<Item = u8> {
func sizeAndPrefixIterator(size uint, prefix uint8, prefixMask int) []byte {
	// let max_value = 255u8 >> prefix_mask;
	maxValue := uint8(255) >> prefixMask
	// let l1 = core::cmp::min((max_value as usize).saturating_sub(1), size);
	l1 := saturating.Sub(maxValue, 1)
	if size < uint(l1) {
		l1 = uint8(size)
	}
	//
	//	let (first_byte, mut rem) = if size == l1 {
	//		(once(prefix + l1 as u8), 0)
	//	} else {
	//
	//		(once(prefix + max_value as u8), size - l1)
	//	};
	var firstByte uint8
	var rem uint
	if size == uint(l1) {
		firstByte = prefix + l1
		rem = 0
	} else {
		firstByte = prefix + maxValue
		rem = size - uint(l1)
	}
	//
	//	let next_bytes = move || {
	//		if rem > 0 {
	//			if rem < 256 {
	//				let result = rem - 1;
	//				rem = 0;
	//				Some(result as u8)
	//			} else {
	//				rem = rem.saturating_sub(255);
	//				Some(255)
	//			}
	//		} else {
	//			None
	//		}
	//	};
	nextBytes := make([]byte, 0)
	for {
		if rem > 0 {
			if rem < 256 {
				result := rem - 1
				rem = 0
				nextBytes = append(nextBytes, uint8(result))
			} else {
				rem = saturating.Sub(rem, 255)
				nextBytes = append(nextBytes, 255)
			}
		} else {
			break
		}
	}

	return append([]byte{firstByte}, nextBytes...)
}
