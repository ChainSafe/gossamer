// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
export function didUpdateToBool(didUpdate, id) {
  return didUpdate.isSome ? didUpdate.unwrap().some(paraId => paraId.eq(id)) : false;
}