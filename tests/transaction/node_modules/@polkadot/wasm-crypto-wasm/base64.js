// Copyright 2019-2021 @polkadot/wasm-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
export function base64Decode(data) {
  const bytes = [];
  let byte = 0;
  let bits = 0;

  for (let i = 0; i < data.length && data[i] !== '='; i++) {
    // each character represents 6 bits
    byte = byte << 6 | chars.indexOf(data[i]); // each byte needs to contain 8 bits

    if ((bits += 6) >= 8) {
      bytes.push(byte >>> (bits -= 8) & 0xff);
    }
  }

  return Uint8Array.from(bytes);
}