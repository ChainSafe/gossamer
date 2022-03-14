// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aToU8a } from '@polkadot/util';
export function findCall(registry, callIndex) {
  return registry.findMetaCall(u8aToU8a(callIndex));
}
export function findError(registry, errorIndex) {
  return registry.findMetaError(u8aToU8a(errorIndex));
}