// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { decodeAddress } from "./decode.js";
export function addressToU8a(who) {
  return decodeAddress(who);
}