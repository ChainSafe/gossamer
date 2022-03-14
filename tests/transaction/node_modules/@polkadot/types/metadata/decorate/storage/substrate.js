// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { createRuntimeFunction } from "./util.js";
const prefix = 'Substrate';
const section = 'substrate';

function createSubstrateFn(method, key, meta) {
  return createRuntimeFunction({
    method,
    prefix,
    section
  }, key, meta);
}

export const substrate = {
  changesTrieConfig: createSubstrateFn('changesTrieConfig', ':changes_trie', {
    docs: 'Changes trie configuration is stored under this key.',
    type: 'u32'
  }),
  childStorageKeyPrefix: createSubstrateFn('childStorageKeyPrefix', ':child_storage:', {
    docs: 'Prefix of child storage keys.',
    type: 'u32'
  }),
  code: createSubstrateFn('code', ':code', {
    docs: 'Wasm code of the runtime.',
    type: 'Bytes'
  }),
  extrinsicIndex: createSubstrateFn('extrinsicIndex', ':extrinsic_index', {
    docs: 'Current extrinsic index (u32) is stored under this key.',
    type: 'u32'
  }),
  heapPages: createSubstrateFn('heapPages', ':heappages', {
    docs: 'Number of wasm linear memory pages required for execution of the runtime.',
    type: 'u64'
  })
};