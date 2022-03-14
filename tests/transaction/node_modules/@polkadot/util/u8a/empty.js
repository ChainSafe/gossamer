// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name u8aEmpty
 * @summary Tests for a `Uint8Array` for emptyness
 * @description
 * Checks to see if the input `Uint8Array` has zero length or contains all 0 values.
 */
export function u8aEmpty(value) {
  // on smaller values < 64 bytes, the byte-by-byte compare is faster than
  // allocating yet another object for DataView (on large buffers the DataView
  // is much faster)
  for (let i = 0; i < value.length; i++) {
    if (value[i]) {
      return false;
    }
  }

  return true;
}