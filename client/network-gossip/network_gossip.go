package gossip

// / Abstraction over a network.
//
//	pub trait Network<B: BlockT>: NetworkPeers + NetworkEventStream + NetworkNotification {
//		fn add_set_reserved(&self, who: PeerId, protocol: ProtocolName) {
//			let addr =
//				iter::once(multiaddr::Protocol::P2p(who.into())).collect::<multiaddr::Multiaddr>();
//			let result = self.add_peers_to_reserved_set(protocol, iter::once(addr).collect());
//			if let Err(err) = result {
//				log::error!(target: "gossip", "add_set_reserved failed: {}", err);
//			}
//		}
//	}
type Network interface{}
