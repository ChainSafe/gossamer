// Copyright 2017-2022 @polkadot/types-codec authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { Text } from "../native/Text.js";
import { sanitize } from "../utils/index.js";
/**
 * @name Type
 * @description
 * This is a extended version of Text, specifically to handle types. Here we rely fully
 * on what Text provides us, however we also adjust the types received from the runtime,
 * i.e. we remove the `T::` prefixes found in some types for consistency across implementation.
 */

export class Type extends Text {
  constructor(registry, value = '') {
    super(registry, value);
    this.setOverride(sanitize(this.toString()));
  }
  /**
   * @description Returns the base runtime type name for this instance
   */


  toRawType() {
    return 'Type';
  }

}