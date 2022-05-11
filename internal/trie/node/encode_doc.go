// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// Each node encoding has a header of one or more bytes.
// The first byte contains the node variant and some
// or all of the partial key length of the node.
// If the partial key length cannot fit in the first byte,
// additional bytes are added to the header to represent
// the total partial key length.
// The header is then concatenated with the partial key,
// encoded as Little Endian bytes.
// The rest concatenated depends on the node variant.
//
// For leaves, the SCALE-encoded leaf value is concatenated.
// For branches, the following is concatenated, where
// `|` denotes concatenation:
// 2 bytes children bitmap | SCALE-encoded branch value |
// Hash(Encoding(Child[0])) | ... | Hash(Encoding(Child[n]))
//
// See https://spec.polkadot.network/#sect-state-storage
// for more details.
