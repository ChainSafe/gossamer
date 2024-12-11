// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

// / Usage statistics for running client instance.
// /
// / Returning backend determines the scope of these stats,
// / but usually it is either from service start or from previous
// / gathering of the statistics.
type UsageInfo struct {
	//		/// Memory statistics.
	//		pub memory: MemoryInfo,
	//		/// I/O statistics.
	//		pub io: IoInfo,
}

// impl fmt::Display for UsageInfo {
// 	fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
// 		write!(
// 			f,
// 			"caches: ({} state, {} db overlay), \
// 			 i/o: ({} tx, {} write, {} read, {} avg tx, {}/{} key cache reads/total, {} trie nodes writes)",
// 			self.memory.state_cache,
// 			self.memory.database_cache,
// 			self.io.transactions,
// 			self.io.bytes_written,
// 			self.io.bytes_read,
// 			self.io.average_transaction_size,
// 			self.io.state_reads_cache,
// 			self.io.state_reads,
// 			self.io.state_writes_nodes,
// 		)
// 	}
// }

// / List of operations to be performed on storage aux data.
// / First tuple element is the encoded data key.
// / Second tuple element is the encoded optional data to write.
// / If `None`, the key and the associated data are deleted from storage.
// pub type AuxDataOperations = Vec<(Vec<u8>, Option<Vec<u8>>)>;
type AuxDataOperation struct {
	Key  []byte
	Data []byte
}
type AuxDataOperations []AuxDataOperation
