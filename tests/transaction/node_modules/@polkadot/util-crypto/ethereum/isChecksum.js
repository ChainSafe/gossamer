// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aToHex } from '@polkadot/util';
import { keccakAsU8a } from "../keccak/index.js";

function isInvalidChar(char, byte) {
  return char !== (byte > 7 ? char.toUpperCase() : char.toLowerCase());
}

export function isEthereumChecksum(_address) {
  const address = _address.replace('0x', '');

  const hash = u8aToHex(keccakAsU8a(address.toLowerCase()), -1, false);

  for (let i = 0; i < 40; i++) {
    if (isInvalidChar(address[i], parseInt(hash[i], 16))) {
      return false;
    }
  }

  return true;
}