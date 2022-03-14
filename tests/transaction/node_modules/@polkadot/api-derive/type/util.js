// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
export function extractAuthor(digest, sessionValidators = []) {
  const [citem] = digest.logs.filter(e => e.isConsensus);
  const [pitem] = digest.logs.filter(e => e.isPreRuntime);
  const [sitem] = digest.logs.filter(e => e.isSeal);
  let accountId;

  try {
    // This is critical to be first for BABE (before Consensus)
    // If not first, we end up dropping the author at session-end
    if (pitem) {
      const [engine, data] = pitem.asPreRuntime;
      accountId = engine.extractAuthor(data, sessionValidators);
    }

    if (!accountId && citem) {
      const [engine, data] = citem.asConsensus;
      accountId = engine.extractAuthor(data, sessionValidators);
    } // SEAL, still used in e.g. Kulupu for pow


    if (!accountId && sitem) {
      const [engine, data] = sitem.asSeal;
      accountId = engine.extractAuthor(data, sessionValidators);
    }
  } catch {// ignore
  }

  return accountId;
}