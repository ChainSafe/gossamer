// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { isFunction } from '@polkadot/util';
import { withSection } from "./helpers.js"; // We are re-exporting these from here to ensure that *.d.ts generation is correct

export function prime(section) {
  return withSection(section, query => () => isFunction(query === null || query === void 0 ? void 0 : query.prime) ? query.prime().pipe(map(o => o.unwrapOr(null))) : of(null));
}