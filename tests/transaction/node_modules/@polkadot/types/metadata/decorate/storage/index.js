// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { lazyMethod, lazyMethods, stringCamelCase } from '@polkadot/util';
import { objectNameToCamel } from "../util.js";
import { createFunction, createKeyRaw, NO_RAW_ARGS } from "./createFunction.js";
import { getStorage } from "./getStorage.js";
import { createRuntimeFunction } from "./util.js";
const VERSION_NAME = 'palletVersion';
const VERSION_KEY = ':__STORAGE_VERSION__:';
const VERSION_DOCS = {
  docs: 'Returns the current pallet version from storage',
  type: 'u16'
};
/** @internal */
// eslint-disable-next-line @typescript-eslint/no-unused-vars

export function decorateStorage(registry, {
  pallets
}, _metaVersion) {
  const result = getStorage(registry);

  for (let i = 0; i < pallets.length; i++) {
    const {
      name,
      storage
    } = pallets[i];

    if (storage.isSome) {
      const section = stringCamelCase(name);
      const {
        items,
        prefix: _prefix
      } = storage.unwrap();

      const prefix = _prefix.toString();

      lazyMethod(result, section, () => lazyMethods({
        palletVersion: createRuntimeFunction({
          method: VERSION_NAME,
          prefix,
          section
        }, createKeyRaw(registry, {
          method: VERSION_KEY,
          prefix: name.toString()
        }, NO_RAW_ARGS), VERSION_DOCS)(registry)
      }, items, meta => createFunction(registry, {
        meta,
        method: meta.name.toString(),
        prefix,
        section
      }, {}), objectNameToCamel));
    }
  }

  return result;
}