// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { substrate } from "./substrate.js";
/** @internal */

export function getStorage(registry) {
  const storage = {};
  const entries = Object.entries(substrate);

  for (let e = 0; e < entries.length; e++) {
    storage[entries[e][0]] = entries[e][1](registry);
  }

  return {
    substrate: storage
  };
}