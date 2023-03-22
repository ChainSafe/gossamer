package parachaininteraction

/*

A node can let us know that it is a collator using `Declare` message
```
	#[derive(Debug, Clone, Encode, Decode, PartialEq, Eq)]
	pub enum CollatorProtocolMessage {
		/// Declare the intent to advertise collations under a collator ID, attaching a
		/// signature of the `PeerId` of the node using the given collator ID key.
		#[codec(index = 0)]
		Declare(CollatorId, ParaId, CollatorSignature),
		/// Advertise a collation to a validator. Can only be sent once the peer has
		/// declared that they are a collator with given ID.
		#[codec(index = 1)]
		AdvertiseCollation(Hash),
		/// A collation sent to a validator was seconded.
		#[codec(index = 4)]
		CollationSeconded(Hash, UncheckedSignedFullStatement),
	}
```

Register two protocols:
- ValidationProtocolV1
- CollationProtocolV1


- Use service.SendMessage to send messages
*/
