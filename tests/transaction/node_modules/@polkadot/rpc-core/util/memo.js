// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { Observable } from 'rxjs';
import { memoize } from '@polkadot/util';
import { drr } from "./drr.js";
// Wraps a derive, doing 2 things to optimize calls -
//   1. creates a memo of the inner fn -> Observable, removing when unsubscribed
//   2. wraps the observable in a drr() (which includes an unsub delay)

/** @internal */
// eslint-disable-next-line @typescript-eslint/ban-types
export function memo(instanceId, inner) {
  const options = {
    getInstanceId: () => instanceId
  };
  const cached = memoize((...params) => new Observable(observer => {
    const subscription = inner(...params).subscribe(observer);
    return () => {
      cached.unmemoize(...params);
      subscription.unsubscribe();
    };
  }).pipe(drr()), options);
  return cached;
}