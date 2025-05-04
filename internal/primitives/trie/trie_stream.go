package trie

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	trieroot "github.com/ChainSafe/gossamer/internal/primitives/trie/trie-root"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// / Codec-flavored TrieStream.
// #[derive(Default, Clone)]
// pub struct TrieStream {
type TrieStream[Hasher hashdb.Hasher[H], H hashdb.Hash] struct {
	/// Current node buffer.
	// 	buffer: Vec<u8>,
	buffer []byte
}

// impl TrieStream {
// 	// useful for debugging but not used otherwise
// 	pub fn as_raw(&self) -> &[u8] {
// 		&self.buffer
// 	}
// }

// use trie_root::Value as TrieStreamValue;
//
//	impl trie_root::TrieStream for TrieStream {
//		fn new() -> Self {
//			Self { buffer: Vec::new() }
//		}
func (ts *TrieStream[Hasher, H]) New() trieroot.TrieStream {
	return NewTrieStream[Hasher, H]()
}
func NewTrieStream[Hasher hashdb.Hasher[H], H hashdb.Hash]() *TrieStream[Hasher, H] {
	return &TrieStream[Hasher, H]{buffer: make([]byte, 0)}
}

//	fn append_empty_data(&mut self) {
//		self.buffer.push(trie_constants::EMPTY_TRIE);
//	}
func (ts *TrieStream[Hasher, H]) AppendEmptyData() {
	ts.buffer = append(ts.buffer, emptyTrie)
}

//	fn append_leaf(&mut self, key: &[u8], value: TrieStreamValue) {
//		let kind = match &value {
//			TrieStreamValue::Inline(..) => NodeKind::Leaf,
//			TrieStreamValue::Node(..) => NodeKind::HashedValueLeaf,
//		};
//		self.buffer.extend(fuse_nibbles_node(key, kind));
//		match &value {
//			TrieStreamValue::Inline(value) => {
//				Compact(value.len() as u32).encode_to(&mut self.buffer);
//				self.buffer.extend_from_slice(value);
//			},
//			TrieStreamValue::Node(hash) => {
//				self.buffer.extend_from_slice(hash.as_slice());
//			},
//		};
//	}
func (ts *TrieStream[Hasher, H]) AppendLeaf(key []byte, value trieroot.Value) {
	var kind nodeKind
	switch value.(type) {
	case trieroot.InlineValue:
		kind = nodeKindLeaf
	case trieroot.NodeValue:
		kind = nodeKindHashedValueLeaf
	default:
		panic("unreachable")
	}
	ts.buffer = append(ts.buffer, fuseNibblesNode(key, kind)...)
	switch value := value.(type) {
	case trieroot.InlineValue:
		length := scale.MustMarshal(len(value))
		ts.buffer = append(ts.buffer, length...)
		ts.buffer = append(ts.buffer, value...)
	case trieroot.NodeValue:
		ts.buffer = append(ts.buffer, value...)
	default:
		panic("unreachable")
	}
}

// 	fn begin_branch(
// 		&mut self,
// 		maybe_partial: Option<&[u8]>,
// 		maybe_value: Option<TrieStreamValue>,
// 		has_children: impl Iterator<Item = bool>,
// 	) {
// 		if let Some(partial) = maybe_partial {
// 			let kind = match &maybe_value {
// 				None => NodeKind::BranchNoValue,
// 				Some(TrieStreamValue::Inline(..)) => NodeKind::BranchWithValue,
// 				Some(TrieStreamValue::Node(..)) => NodeKind::HashedValueBranch,
// 			};

//			self.buffer.extend(fuse_nibbles_node(partial, kind));
//			let bm = branch_node_bit_mask(has_children);
//			self.buffer.extend([bm.0, bm.1].iter());
//		} else {
//			unreachable!("trie stream codec only for no extension trie");
//		}
//		match maybe_value {
//			None => (),
//			Some(TrieStreamValue::Inline(value)) => {
//				Compact(value.len() as u32).encode_to(&mut self.buffer);
//				self.buffer.extend_from_slice(value);
//			},
//			Some(TrieStreamValue::Node(hash)) => {
//				self.buffer.extend_from_slice(hash.as_slice());
//			},
//		}
//	}
func (ts *TrieStream[Hasher, H]) BeginBranch(maybePartial []byte, maybeValue trieroot.Value, hasChildren []bool) {
	if maybePartial != nil {
		var kind nodeKind
		if maybeValue == nil {
			kind = nodeKindBranchNoValue
		} else {
			switch maybeValue.(type) {
			case trieroot.InlineValue:
				kind = nodeKindBranchWithValue
			case trieroot.NodeValue:
				kind = nodeKindHashedValueBranch
			default:
				panic("unreachable")
			}
		}

		ts.buffer = append(ts.buffer, fuseNibblesNode(maybePartial, kind)...)
		a, b := branchNodeBitMask(hasChildren)
		ts.buffer = append(ts.buffer, a, b)
	} else {
		panic("trie stream codec only for no extension trie")
	}
	if maybeValue == nil {
		return
	}
	switch value := maybeValue.(type) {
	case trieroot.InlineValue:
		length := scale.MustMarshal(len(value))
		ts.buffer = append(ts.buffer, length...)
		ts.buffer = append(ts.buffer, value...)
	case trieroot.NodeValue:
		ts.buffer = append(ts.buffer, value...)
	default:
		panic("unreachable")
	}
}

//	fn append_extension(&mut self, _key: &[u8]) {
//		debug_assert!(false, "trie stream codec only for no extension trie");
//	}
func (ts *TrieStream[Hasher, H]) AppendExtension(key []byte) {
	panic("trie stream codec only for no extension trie")
}

//	fn append_substream<H: Hasher>(&mut self, other: Self) {
//		let data = other.out();
//		match data.len() {
//			0..=31 => data.encode_to(&mut self.buffer),
//			_ => H::hash(&data).as_ref().encode_to(&mut self.buffer),
//		}
//	}
func (ts *TrieStream[Hasher, H]) AppendSubstream(other trieroot.TrieStream) {
	data := other.Out()
	if len(data) <= 31 {
		ts.buffer = append(ts.buffer, scale.MustMarshal(data)...)
	} else {
		hash := (*new(Hasher)).Hash(data)
		ts.buffer = append(ts.buffer, scale.MustMarshal(hash.Bytes())...)
	}
}

//		fn out(self) -> Vec<u8> {
//			self.buffer
//		}
//	}
func (ts *TrieStream[Hasher, H]) Out() []byte {
	return ts.buffer
}

func (ts *TrieStream[Hasher, H]) AppendEmptyChild()              {}
func (ts *TrieStream[Hasher, H]) EndBranch(value trieroot.Value) {}

// fn branch_node_bit_mask(has_children: impl Iterator<Item = bool>) -> (u8, u8) {
func branchNodeBitMask(hasChildren []bool) (uint8, uint8) {
	// let mut bitmap: u16 = 0;
	// let mut cursor: u16 = 1;
	var (
		bitmap uint16
		cursor uint16 = 1
	)
	//
	//	for v in has_children {
	//		if v {
	//			bitmap |= cursor
	//		}
	//		cursor <<= 1;
	//	}
	for _, v := range hasChildren {
		if v {
			bitmap |= cursor
		}
		cursor <<= 1
	}
	//
	// ((bitmap % 256) as u8, (bitmap / 256) as u8)
	return uint8(bitmap % 256), uint8(bitmap / 256)
}

// /// Create a leaf/branch node, encoding a number of nibbles.
// fn fuse_nibbles_node(nibbles: &[u8], kind: NodeKind) -> impl Iterator<Item = u8> + '_ {
func fuseNibblesNode(nibbles []byte, kind nodeKind) []byte {
	// let size = nibbles.len();
	size := uint(len(nibbles))
	//
	//	let iter_start = match kind {
	//		NodeKind::Leaf => size_and_prefix_iterator(size, trie_constants::LEAF_PREFIX_MASK, 2),
	//		NodeKind::BranchNoValue =>
	//			size_and_prefix_iterator(size, trie_constants::BRANCH_WITHOUT_MASK, 2),
	//		NodeKind::BranchWithValue =>
	//			size_and_prefix_iterator(size, trie_constants::BRANCH_WITH_MASK, 2),
	//		NodeKind::HashedValueLeaf =>
	//			size_and_prefix_iterator(size, trie_constants::ALT_HASHING_LEAF_PREFIX_MASK, 3),
	//		NodeKind::HashedValueBranch =>
	//			size_and_prefix_iterator(size, trie_constants::ALT_HASHING_BRANCH_WITH_MASK, 4),
	//	};
	var start []byte
	switch kind {
	case nodeKindLeaf:
		start = sizeAndPrefixIterator(size, leafPrefixMask, 2)
	case nodeKindBranchNoValue:
		start = sizeAndPrefixIterator(size, branchWithoutMask, 2)
	case nodeKindBranchWithValue:
		start = sizeAndPrefixIterator(size, branchWithMask, 2)
	case nodeKindHashedValueLeaf:
		start = sizeAndPrefixIterator(size, altHashingLeafPrefixMask, 3)
	case nodeKindHashedValueBranch:
		start = sizeAndPrefixIterator(size, altHashingBranchWithMask, 4)
	}
	//
	// iter_start
	//
	//	.chain(if nibbles.len() % 2 == 1 { Some(nibbles[0]) } else { None })
	//	.chain(nibbles[nibbles.len() % 2..].chunks(2).map(|ch| ch[0] << 4 | ch[1]))
	if len(nibbles)%2 == 1 {
		start = append(start, nibbles[0])
	}
	begin := len(nibbles) % 2
	for i := begin; i < len(nibbles); i += 2 {
		start = append(start, nibbles[i]<<4|nibbles[i+1])
	}
	return start
}
