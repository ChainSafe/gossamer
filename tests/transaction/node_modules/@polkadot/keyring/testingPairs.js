// Copyright 2017-2022 @polkadot/keyring authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { nobody } from "./pair/nobody.js";
import { createTestKeyring } from "./testing.js";
export function createTestPairs(options, isDerived = true) {
  const keyring = createTestKeyring(options, isDerived);
  const pairs = keyring.getPairs();
  const map = {
    nobody: nobody()
  };

  for (const p of pairs) {
    map[p.meta.name] = p;
  }

  return map;
}