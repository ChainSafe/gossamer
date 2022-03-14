// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { validateAddress } from "./validate.js";
export function isAddress(address, ignoreChecksum, ss58Format) {
  try {
    return validateAddress(address, ignoreChecksum, ss58Format);
  } catch (error) {
    return false;
  }
}