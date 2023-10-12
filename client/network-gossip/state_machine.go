package gossip

/// Consensus network protocol handler. Manages statements and candidate requests.
// pub struct ConsensusGossip<B: BlockT> {
// 	peers: HashMap<PeerId, PeerConsensus<B::Hash>>,
// 	messages: Vec<MessageEntry<B>>,
// 	known_messages: LruCache<B::Hash, ()>,
// 	protocol: ProtocolName,
// 	validator: Arc<dyn Validator<B>>,
// 	next_broadcast: Instant,
// 	metrics: Option<Metrics>,
// }

type ConsensusGossip struct {
}
