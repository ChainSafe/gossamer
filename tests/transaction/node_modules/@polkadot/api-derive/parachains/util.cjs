"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.didUpdateToBool = didUpdateToBool;

// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
function didUpdateToBool(didUpdate, id) {
  return didUpdate.isSome ? didUpdate.unwrap().some(paraId => paraId.eq(id)) : false;
}