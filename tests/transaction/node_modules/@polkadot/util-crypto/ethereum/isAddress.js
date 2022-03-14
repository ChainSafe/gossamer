// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isHex } from '@polkadot/util';
import { isEthereumChecksum } from "./isChecksum.js";
export function isEthereumAddress(address) {
  if (!address || address.length !== 42 || !isHex(address)) {
    return false;
  } else if (/^(0x)?[0-9a-f]{40}$/.test(address) || /^(0x)?[0-9A-F]{40}$/.test(address)) {
    return true;
  }

  return isEthereumChecksum(address);
}