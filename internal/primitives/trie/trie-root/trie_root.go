package trieroot

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/kv"
	"github.com/tidwall/btree"
)

type KeyValue = kv.KeyValue

// / Different possible value to use for node encoding.
// #[derive(Clone)]
//
//	pub enum Value<'a> {
//		/// Contains a full value.
//		Inline(&'a [u8]),
//		/// Contains hash of a value.
//		Node(Vec<u8>),
//	}
type Value interface {
	isValue()
}
type (
	/// Contains a full value.
	InlineValue []byte
	/// Contains hash of a value.
	NodeValue []byte
)

func (InlineValue) isValue() {}
func (NodeValue) isValue()   {}

// fn new<H: Hasher>(value: &'a [u8], threshold: Option<u32>) -> Value<'a> {
func NewValue[Hasher hashdb.Hasher[H], H hashdb.Hash](value []byte, threshold *uint32) Value {
	//	if let Some(threshold) = threshold {
	//		if value.len() >= threshold as usize {
	//			Value::Node(H::hash(value).as_ref().to_vec())
	//		} else {
	//			Value::Inline(value)
	//		}
	//	} else {
	//
	//		Value::Inline(value)
	//	}
	if threshold != nil && len(value) >= int(*threshold) {
		h := (*new(Hasher)).Hash(value)
		return NodeValue(h.Bytes())
	}
	return InlineValue(value)
}

// / Byte-stream oriented trait for constructing closed-form tries.
// pub trait TrieStream {
type TrieStream interface {
	/// Construct a new `TrieStream`
	// fn new() -> Self;
	New() TrieStream
	/// Append an Empty node
	AppendEmptyData()
	/// Start a new Branch node, possibly with a value; takes a list indicating
	/// which slots in the Branch node has further child nodes.
	BeginBranch(
		maybeKey []byte,
		maybeValue Value,
		hasChildren []bool,
	)
	/// Append an empty child node. Optional.
	AppendEmptyChild()
	/// Wrap up a Branch node portion of a `TrieStream` and append the value
	/// stored on the Branch (if any).
	EndBranch(value Value)
	/// Append a Leaf node
	// fn append_leaf(&mut self, key: &[u8], value: Value);
	AppendLeaf(key []byte, value Value)
	// /// Append an Extension node
	// fn append_extension(&mut self, key: &[u8]);
	AppendExtension(key []byte)
	// /// Append a Branch of Extension substream
	// fn append_substream<H: Hasher>(&mut self, other: Self);
	AppendSubstream(other TrieStream)
	/// Return the finished `TrieStream` as a vector of bytes.
	Out() []byte
}

// fn shared_prefix_length<T: Eq>(first: &[T], second: &[T]) -> usize {
func sharedPrefixLength(first []byte, second []byte) uint {
	// first
	//
	//	.iter()
	//	.zip(second.iter())
	//	.position(|(f, s)| f != s)
	//	.unwrap_or_else(|| cmp::min(first.len(), second.len()))
	length := len(first)
	if len(second) < length {
		length = len(second)
	}
	var position int = -1
	for i := 0; i < length; i++ {
		var (
			a byte
			b byte
		)
		if i < len(first) {
			a = first[i]
		}
		if i < len(second) {
			b = second[i]
		}
		if a != b {
			position = i
			break
		}
	}
	if position == -1 {
		position = len(first)
		if len(second) < position {
			position = len(second)
		}
	}
	return uint(position)
}

// fn trie_root_inner<H, S, I, A, B>(input: I, no_extension: bool, threshold: Option<u32>) -> H::Out
// where
//
//	I: IntoIterator<Item = (A, B)>,
//	A: AsRef<[u8]> + Ord,
//	B: AsRef<[u8]>,
//	H: Hasher,
//	S: TrieStream,
//
// {
func TrieRoot[Hasher hashdb.Hasher[H], H hashdb.Hash](
	input []KeyValue,
	threshold *uint32,
	stream TrieStream,
) H {

	// 	// first put elements into btree to sort them and to remove duplicates
	// 	let input = input.into_iter().collect::<BTreeMap<_, _>>();
	inputMap := btree.Map[string, []byte]{}
	for _, kv := range input {
		inputMap.Set(string(kv.Key), kv.Value)
	}

	// 	// convert to nibbles
	// 	let mut nibbles = Vec::with_capacity(input.keys().map(|k| k.as_ref().len()).sum::<usize>() * 2);
	// 	let mut lens = Vec::with_capacity(input.len() + 1);
	// 	lens.push(0);
	// 	for k in input.keys() {
	// 		for &b in k.as_ref() {
	// 			nibbles.push(b >> 4);
	// 			nibbles.push(b & 0x0F);
	// 		}
	// 		lens.push(nibbles.len());
	// 	}
	nibblesLen := 0
	for _, kv := range input {
		nibblesLen += len(kv.Key) * 2
	}
	nibbles := make([]byte, 0, nibblesLen)
	lens := make([]uint, 0, len(input)+1)
	lens = append(lens, 0)
	for _, kv := range input {
		for _, b := range kv.Key {
			nibbles = append(nibbles, b>>4)
			nibbles = append(nibbles, b&0x0F)
		}
		lens = append(lens, uint(len(nibbles)))
	}

	// 	// then move them to a vector
	// 	let input = input
	// 		.into_iter()
	// 		.zip(lens.windows(2))
	// 		.map(|((_, v), w)| (&nibbles[w[0]..w[1]], v))
	// 		.collect::<Vec<_>>();
	trieInput := make([]KeyValue, len(input))
	for i, kv := range input {
		in := KeyValue{
			Key:   nibbles[lens[i]:lens[i+1]],
			Value: kv.Value,
		}
		trieInput[i] = in
	}

	// let mut stream = S::new();

	// build_trie::<H, S, _, _>(&input, 0, &mut stream, no_extension, threshold);
	buildTrie[Hasher, H](trieInput, 0, stream, threshold)
	// H::hash(&stream.out())
	return (*new(Hasher)).Hash(stream.Out())
}

func buildTrie[Hasher hashdb.Hasher[H], H hashdb.Hash](
	input []KeyValue,
	cursor uint,
	stream TrieStream,
	threshold *uint32,
) {
	// match input.len() {
	switch len(input) {
	case 0:
		// No input, just append empty data.
		// 	0 => stream.append_empty_data(),
		stream.AppendEmptyData()

	case 1:
		// Leaf node; append the remainder of the key and the value. Done.
		// 		let value = Value::new::<H>(input[0].1.as_ref(), threshold);
		// 		stream.append_leaf(&input[0].0.as_ref()[cursor..], value)
		value := NewValue[Hasher](input[0].Value, threshold)
		stream.AppendLeaf(input[0].Key[cursor:], value)

		// 	_ => {
	default:
		// We have multiple items in the input. Figure out if we should add an
		// extension node or a branch node.
		// 		let (key, value) = (&input[0].0.as_ref(), input[0].1.as_ref());
		key := input[0].Key
		value := input[0].Value
		// Count the number of nibbles in the other elements that are
		// shared with the first key.
		// e.g. input = [ [1'7'3'10'12'13], [1'7'3'], [1'7'7'8'9'] ] => [1'7'] is common => 2
		// 		let shared_nibble_count = input.iter().skip(1).fold(key.len(), |acc, &(ref k, _)| {
		// 			cmp::min(shared_prefix_length(key, k.as_ref()), acc)
		// 		});
		sharedNibbleCount := uint(len(key))
		for _, kv := range input[1:] {
			sharedPrefixLength := sharedPrefixLength(key, kv.Key)
			if sharedPrefixLength < sharedNibbleCount {
				sharedNibbleCount = sharedPrefixLength
			}
		}
		// Add an extension node if the number of shared nibbles is greater
		// than what we saw on the last call (`cursor`): append the new part
		// of the path then recursively append the remainder of all items
		// who had this partial key.
		// 		let (cursor, o_branch_slice) = if no_extension {
		// 			if shared_nibble_count > cursor {
		// 				(shared_nibble_count, Some(&key[cursor..shared_nibble_count]))
		// 			} else {
		// 				(cursor, Some(&key[0..0]))
		// 			}
		// 		} else if shared_nibble_count > cursor {
		// 			stream.append_extension(&key[cursor..shared_nibble_count]);
		// 			build_trie_trampoline::<H, _, _, _>(
		// 				input,
		// 				shared_nibble_count,
		// 				stream,
		// 				no_extension,
		// 				threshold,
		// 			);
		// 			return
		// 		} else {
		// 			(cursor, None)
		// 		};

		var oBranchSlice []byte

		if sharedNibbleCount > cursor {
			oBranchSlice = key[cursor:sharedNibbleCount]
			cursor = sharedNibbleCount
		} else {
			cursor = cursor
			oBranchSlice = key[0:0]
		}

		// We'll be adding a branch node because the path is as long as it gets.
		// First we need to figure out what entries this branch node will have...

		// We have a a value for exactly this key. Branch node will have a value
		// attached to it.
		// 		let value = if cursor == key.len() { Some(value) } else { None };
		var val []byte
		if cursor == uint(len(key)) {
			val = value
		} else {
			val = nil
		}

		// We need to know how many key nibbles each of the children account for.
		// 		let mut shared_nibble_counts = [0usize; 16];
		sharedNibblesCounts := make([]uint, 16)
		{
			// If the Branch node has a value then the first of the input keys
			// is exactly the key for that value and we don't care about it
			// when finding shared nibbles for our child nodes. (We know it's
			// the first of the input keys, because the input is sorted)
			// 			let mut begin = match value {
			// 				None => 0,
			// 				_ => 1,
			// 			};
			// 			for i in 0..16 {
			// 				shared_nibble_counts[i] = input[begin..]
			// 					.iter()
			// 					.take_while(|(k, _)| k.as_ref()[cursor] == i as u8)
			// 					.count();
			// 				begin += shared_nibble_counts[i];
			// 			}
			var begin uint
			if val == nil {
				begin = 0
			} else {
				begin = 1
			}
			for i := uint(0); i < 16; i++ {
				var sharedNibbleCount uint
				for _, kv := range input[begin:] {
					if kv.Key[cursor] == byte(i) {
						sharedNibbleCount++
					} else {
						break
					}
				}
				sharedNibblesCounts[i] = sharedNibbleCount
				begin += sharedNibbleCount
			}
		}

		// Put out the node header:
		// 		let value = value.map(|v| Value::new::<H>(v, threshold));
		// 		stream.begin_branch(
		// 			o_branch_slice,
		// 			value.clone(),
		// 			shared_nibble_counts.iter().map(|&n| n > 0),
		// 		);
		var v Value
		if val != nil {
			v = NewValue[Hasher](val, threshold)
		}
		hasChildren := make([]bool, 16)
		for i, count := range sharedNibblesCounts {
			if count > 0 {
				hasChildren[i] = true
			}
		}
		stream.BeginBranch(oBranchSlice, v, hasChildren)

		// Fill in each slot in the branch node. We don't need to bother with empty slots
		// since they were registered in the header.
		// 		let mut begin = match value {
		// 			None => 0,
		// 			_ => 1,
		// 		};
		var begin uint = 0
		if val != nil {
			begin = 1
		}
		// 		for &count in &shared_nibble_counts {
		// 			if count > 0 {
		// 				build_trie_trampoline::<H, S, _, _>(
		// 					&input[begin..(begin + count)],
		// 					cursor + 1,
		// 					stream,
		// 					no_extension,
		// 					threshold.clone(),
		// 				);
		// 				begin += count;
		// 			} else {
		// 				stream.append_empty_child();
		// 			}
		// 		}
		for _, count := range sharedNibblesCounts {
			if count > 0 {
				buildTrieTrampoline[Hasher, H](
					input[begin:begin+count],
					cursor+1,
					stream,
					threshold,
				)
				begin += count
			} else {
				stream.AppendEmptyChild()
			}
		}

		// 		stream.end_branch(value);
		stream.EndBranch(v)
	}
}

// fn build_trie_trampoline<H, S, A, B>(
//
//	input: &[(A, B)],
//	cursor: usize,
//	stream: &mut S,
//	no_extension: bool,
//	threshold: Option<u32>,
//
// ) where
//
//	A: AsRef<[u8]>,
//	B: AsRef<[u8]>,
//	H: Hasher,
//	S: TrieStream,
//
// {
func buildTrieTrampoline[Hasher hashdb.Hasher[H], H hashdb.Hash](
	input []KeyValue,
	cursor uint,
	stream TrieStream,
	threshold *uint32,
) {
	// let mut substream = S::new();
	substream := stream.New()
	// build_trie::<H, _, _, _>(input, cursor, &mut substream, no_extension, threshold);
	buildTrie[Hasher, H](input, cursor, substream, threshold)
	// stream.append_substream::<H>(substream);
	stream.AppendSubstream(substream)
}
