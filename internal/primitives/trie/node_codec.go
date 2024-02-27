package trie

// / Concrete implementation of a [`NodeCodecT`] with SCALE encoding.
// /
// / It is generic over `H` the [`Hasher`].
type NodeCodec[H any] struct{}
