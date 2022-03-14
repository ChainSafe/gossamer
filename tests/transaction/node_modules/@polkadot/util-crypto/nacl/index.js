// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @summary Implements [NaCl](http://nacl.cr.yp.to/) secret-key authenticated encryption, public-key authenticated encryption
 */
export { naclDecrypt } from "./decrypt.js";
export { naclEncrypt } from "./encrypt.js";
export { naclBoxPairFromSecret } from "./box/fromSecret.js";
export { naclOpen } from "./open.js";
export { naclSeal } from "./seal.js";