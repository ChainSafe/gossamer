// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { decodeAddress } from "./decode.js";
export function validateAddress(encoded, ignoreChecksum, ss58Format) {
  return !!decodeAddress(encoded, ignoreChecksum, ss58Format);
}