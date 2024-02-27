package recorder

// / The trie recorder.
// /
// / It can be used to record accesses to the trie and then to convert them into a [`StorageProof`].
// pub struct Recorder<H: Hasher> {
type Recorder[H any] struct {
	// inner: Arc<Mutex<RecorderInner<H::Out>>>,
	// /// The estimated encoded size of the storage proof this recorder will produce.
	// ///
	// /// We store this in an atomic to be able to fetch the value while the `inner` is may locked.
	// encoded_size_estimation: Arc<AtomicUsize>,
}
