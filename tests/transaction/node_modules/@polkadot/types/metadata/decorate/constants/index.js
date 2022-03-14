// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { hexToU8a, lazyMethod, lazyMethods, stringCamelCase } from '@polkadot/util';
import { objectNameToCamel } from "../util.js";
/** @internal */
// eslint-disable-next-line @typescript-eslint/no-unused-vars

export function decorateConstants(registry, {
  pallets
}, _version) {
  const result = {};

  for (let i = 0; i < pallets.length; i++) {
    const {
      constants,
      name
    } = pallets[i];

    if (!constants.isEmpty) {
      lazyMethod(result, stringCamelCase(name), () => lazyMethods({}, constants, constant => {
        const codec = registry.createTypeUnsafe(registry.createLookupType(constant.type), [hexToU8a(constant.value.toHex())]);
        codec.meta = constant;
        return codec;
      }, objectNameToCamel));
    }
  }

  return result;
}