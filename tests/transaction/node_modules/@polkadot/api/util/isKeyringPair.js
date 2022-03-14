// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isFunction } from '@polkadot/util';
export function isKeyringPair(account) {
  return isFunction(account.sign);
}