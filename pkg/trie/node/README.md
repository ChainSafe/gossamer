# Trie node

Package node defines the `Node` structure with methods to be used in the modified Merkle-Patricia Radix-16 trie.

## Codec

The following sub-sections precise the encoding of a node.
This encoding is formally described in [the Polkadot specification](https://spec.polkadot.network/#sect-state-storage).

### Header

Each node encoding has a header of one or more bytes.
The first byte contains the node variant and some or all of the partial key length of the node.
If the partial key length cannot fit in the first byte, additional bytes are added to the header to represent the total partial key length.

### Partial key

The header is then concatenated with the partial key of the node, encoded as Little Endian bytes.

### Remaining bytes

The remaining bytes appended depend on the node variant.

- For leaves, the SCALE-encoded leaf storage value is appended.
- For branches, the following elements are concatenated in this order and appended to the previous header+partial key:
  - Children bitmap (2 bytes)
  - SCALE-encoded node storage value
  - Hash(Encoding(Child[0]))
  - Hash(Encoding(Child[1]))
  - ...
  - Hash(Encoding(Child[15]))
