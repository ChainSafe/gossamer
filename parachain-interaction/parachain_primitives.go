package parachaininteraction

// Useful structs defined in cumulus.
// We may never use these directly, but use there scale encoding.
// Useful for getting right test data and reference.

// /// Head data for this parachain.
// #[derive(Default, Clone, Hash, Eq, PartialEq, Encode, Decode, Debug)]
// pub struct HeadData {
// 	/// Block number
// 	pub number: u64,
// 	/// parent block keccak256
// 	pub parent_hash: [u8; 32],
// 	/// hash of post-execution state.
// 	pub post_state: [u8; 32],
// }

// impl HeadData {
// 	pub fn hash(&self) -> [u8; 32] {
// 		keccak256(&self.encode())
// 	}
// }

// /// Block data for this parachain.
// #[derive(Default, Clone, Encode, Decode, Debug)]
// pub struct BlockData {
// 	/// State to begin from.
// 	pub state: u64,
// 	/// Amount to add (wrapping)
// 	pub add: u64,
// }

type HeadData struct {
	Number     uint64
	ParentHash [32]byte
	PostState  [32]byte
}

type BlockData struct {
	State uint64
	Add   uint64
}
