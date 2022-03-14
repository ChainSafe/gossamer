// Copyright 2017-2022 @polkadot/types-codec authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isFunction } from '@polkadot/util';
export function hasEq(o) {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
  return isFunction(o.eq);
}