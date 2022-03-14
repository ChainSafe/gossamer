// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { lazyDeriveSection } from '@polkadot/api-derive';

/**
 * This is a section decorator which keeps all type information.
 */
export function decorateDeriveSections(decorateMethod, derives) {
  const getKeys = s => Object.keys(derives[s]);

  const creator = (s, m) => decorateMethod(derives[s][m]);

  const result = {};
  const names = Object.keys(derives);

  for (let i = 0; i < names.length; i++) {
    lazyDeriveSection(result, names[i], getKeys, creator);
  }

  return result;
}