package service

// / Provides an ability to set a fork sync request for a particular block.
//
//	pub trait NetworkSyncForkRequest<BlockHash, BlockNumber> {
//		/// Notifies the sync service to try and sync the given block from the given
//		/// peers.
//		///
//		/// If the given vector of peers is empty then the underlying implementation
//		/// should make a best effort to fetch the block from any peers it is
//		/// connected to (NOTE: this assumption will change in the future #3629).
//		fn set_sync_fork_request(&self, peers: Vec<PeerId>, hash: BlockHash, number: BlockNumber);
//	}
type NetworkSyncForkRequest interface{}

/// Provides ability to announce blocks to the network.
//
// pub trait NetworkBlock<BlockHash, BlockNumber> {
// 	/// Make sure an important block is propagated to peers.
// 	///
// 	/// In chain-based consensus, we often need to make sure non-best forks are
// 	/// at least temporarily synced. This function forces such an announcement.
// 	fn announce_block(&self, hash: BlockHash, data: Option<Vec<u8>>);

//		/// Inform the network service about new best imported block.
//		fn new_best_block_imported(&self, hash: BlockHash, number: BlockNumber);
//	}
type NetworkBlock interface{}

//	pub trait SyncEventStream: Send + Sync {
//		/// Subscribe to syncing-related events.
//		fn event_stream(&self, name: &'static str) -> Pin<Box<dyn Stream<Item = SyncEvent> + Send>>;
//	}
type SyncEventStream interface{}
