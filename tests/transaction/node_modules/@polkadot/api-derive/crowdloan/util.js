// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
export function extractContributed(paraId, events) {
  var _events$createdAtHash;

  const added = [];
  const removed = [];
  return events.filter(({
    event: {
      data: [, eventParaId],
      method,
      section
    }
  }) => section === 'crowdloan' && ['Contributed', 'Withdrew'].includes(method) && eventParaId.eq(paraId)).reduce((result, {
    event: {
      data: [accountId],
      method
    }
  }) => {
    if (method === 'Contributed') {
      result.added.push(accountId.toHex());
    } else {
      result.removed.push(accountId.toHex());
    }

    return result;
  }, {
    added,
    blockHash: ((_events$createdAtHash = events.createdAtHash) === null || _events$createdAtHash === void 0 ? void 0 : _events$createdAtHash.toHex()) || '-',
    removed
  });
}