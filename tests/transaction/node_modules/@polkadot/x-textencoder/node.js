// Copyright 2017-2022 @polkadot/x-textencoder authors & contributors
// SPDX-License-Identifier: Apache-2.0
import util from 'util';
import { extractGlobal } from '@polkadot/x-global';
export { packageInfo } from "./packageInfo.js";

class Fallback {
  #encoder;

  constructor() {
    this.#encoder = new util.TextEncoder();
  } // For a Jest 26.0.1 environment, Buffer !== Uint8Array


  encode(value) {
    return Uint8Array.from(this.#encoder.encode(value));
  }

}

export const TextEncoder = extractGlobal('TextEncoder', Fallback);