package stats

// / Accumulated usage statistics specific to state machine
// / crate.
type StateMachineStats struct {
	/// Number of read query from runtime
	/// that hit a modified value (in state
	/// machine overlay).
	// pub reads_modified: RefCell<u64>,
	ReadsModified uint64
	/// Size in byte of read queries that
	/// hit a modified value.
	// pub bytes_read_modified: RefCell<u64>,
	BytesReadModified uint64
	/// Number of time a write operation
	/// occurs into the state machine overlay.
	// pub writes_overlay: RefCell<u64>,
	WritesOverlay uint64
	/// Size in bytes of the writes overlay
	/// operation.
	// pub bytes_writes_overlay: RefCell<u64>,
	BytesWritesOverlay uint64
}

// / Usage statistics for state backend.
type UsageInfo struct {
	/// Read statistics (total).
	// pub reads: UsageUnit,
	/// Write statistics (total).
	// pub writes: UsageUnit,
	/// Write trie nodes statistics.
	// pub nodes_writes: UsageUnit,
	/// Write into cached state machine
	/// change overlay.
	// pub overlay_writes: UsageUnit,
	/// Removed trie nodes statistics.
	// pub removed_nodes: UsageUnit,
	/// Cache read statistics.
	// pub cache_reads: UsageUnit,
	/// Modified value read statistics.
	// pub modified_reads: UsageUnit,
	/// Memory used.
	// pub memory: usize,

	/// Moment at which current statistics has been started being collected.
	// pub started: Instant,
	/// Timespan of the statistics.
	// pub span: Duration,
}
