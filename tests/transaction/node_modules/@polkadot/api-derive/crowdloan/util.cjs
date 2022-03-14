"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.extractContributed = extractContributed;

// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
function extractContributed(paraId, events) {
  var _events$createdAtHash;

  const added = [];
  const removed = [];
  return events.filter(_ref => {
    let {
      event: {
        data: [, eventParaId],
        method,
        section
      }
    } = _ref;
    return section === 'crowdloan' && ['Contributed', 'Withdrew'].includes(method) && eventParaId.eq(paraId);
  }).reduce((result, _ref2) => {
    let {
      event: {
        data: [accountId],
        method
      }
    } = _ref2;

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