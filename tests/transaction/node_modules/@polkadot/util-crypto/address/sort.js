// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aSorted } from '@polkadot/util';
import { encodeAddress } from "./encode.js";
import { addressToU8a } from "./util.js";
export function sortAddresses(addresses, ss58Format) {
  const u8aToAddress = u8a => encodeAddress(u8a, ss58Format);

  return u8aSorted(addresses.map(addressToU8a)).map(u8aToAddress);
}