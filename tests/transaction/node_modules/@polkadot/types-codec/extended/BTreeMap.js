// Copyright 2017-2022 @polkadot/types-codec authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { CodecMap } from "./Map.js";
export class BTreeMap extends CodecMap {
  static with(keyType, valType) {
    return class extends BTreeMap {
      constructor(registry, value) {
        super(registry, keyType, valType, value, 'BTreeMap');
      }

    };
  }

}