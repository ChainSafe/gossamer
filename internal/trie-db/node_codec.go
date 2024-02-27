package triedb

// / Trait for trie node encoding/decoding.
// / Uses a type parameter to allow registering
// / positions without colling decode plan.
// pub trait NodeCodec: Sized {
type NodeCodec[HashOut comparable] interface {
	// /// Escape header byte sequence to indicate next node is a
	// /// branch or leaf with hash of value, followed by the value node.
	// const ESCAPE_HEADER: Option<u8> = None;

	// /// Codec error type.
	// type Error: Error;

	// /// Output type of encoded node hasher.
	// type HashOut: AsRef<[u8]>
	// 	+ AsMut<[u8]>
	// 	+ Default
	// 	+ MaybeDebug
	// 	+ PartialEq
	// 	+ Eq
	// 	+ hash::Hash
	// 	+ Send
	// 	+ Sync
	// 	+ Clone
	// 	+ Copy;

	// /// Get the hashed null node.
	// fn hashed_null_node() -> Self::HashOut;

	// /// Decode bytes to a `NodePlan`. Returns `Self::E` on failure.
	// fn decode_plan(data: &[u8]) -> Result<NodePlan, Self::Error>;

	// /// Decode bytes to a `Node`. Returns `Self::E` on failure.
	// fn decode<'a>(data: &'a [u8]) -> Result<Node<'a>, Self::Error> {
	// 	Ok(Self::decode_plan(data)?.build(data))
	// }

	// /// Check if the provided bytes correspond to the codecs "empty" node.
	// fn is_empty_node(data: &[u8]) -> bool;

	// /// Returns an encoded empty node.
	// fn empty_node() -> &'static [u8];

	// /// Returns an encoded leaf node
	// ///
	// /// Note that number_nibble is the number of element of the iterator
	// /// it can possibly be obtain by `Iterator` `size_hint`, but
	// /// for simplicity it is used directly as a parameter.
	// fn leaf_node(partial: impl Iterator<Item = u8>, number_nibble: usize, value: Value) -> Vec<u8>;

	// /// Returns an encoded extension node
	// ///
	// /// Note that number_nibble is the number of element of the iterator
	// /// it can possibly be obtain by `Iterator` `size_hint`, but
	// /// for simplicity it is used directly as a parameter.
	// fn extension_node(
	// 	partial: impl Iterator<Item = u8>,
	// 	number_nibble: usize,
	// 	child_ref: ChildReference<Self::HashOut>,
	// ) -> Vec<u8>;

	// /// Returns an encoded branch node.
	// /// Takes an iterator yielding `ChildReference<Self::HashOut>` and an optional value.
	// fn branch_node(
	// 	children: impl Iterator<Item = impl Borrow<Option<ChildReference<Self::HashOut>>>>,
	// 	value: Option<Value>,
	// ) -> Vec<u8>;

	// /// Returns an encoded branch node with a possible partial path.
	// /// `number_nibble` is the partial path length as in `extension_node`.
	// fn branch_node_nibbled(
	// 	partial: impl Iterator<Item = u8>,
	// 	number_nibble: usize,
	// 	children: impl Iterator<Item = impl Borrow<Option<ChildReference<Self::HashOut>>>>,
	// 	value: Option<Value>,
	// ) -> Vec<u8>;
}
