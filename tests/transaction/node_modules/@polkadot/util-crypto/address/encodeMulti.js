// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { encodeAddress } from "./encode.js";
import { createKeyMulti } from "./keyMulti.js";
/**
 * @name encodeMultiAddress
 * @summary Creates a multisig address.
 * @description
 * Creates a Substrate multisig address based on the input address and the required threshold.
 */

export function encodeMultiAddress(who, threshold, ss58Format) {
  return encodeAddress(createKeyMulti(who, threshold), ss58Format);
}