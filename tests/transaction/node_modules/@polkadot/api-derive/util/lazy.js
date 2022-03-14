// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { lazyMethod, lazyMethods } from '@polkadot/util';
export function lazyDeriveSection(result, section, getKeys, creator) {
  lazyMethod(result, section, () => lazyMethods({}, getKeys(section), method => creator(section, method)));
}