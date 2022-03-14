// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of, switchMap } from 'rxjs';
import { memo } from "../util/index.js";

function withImage(api, nextOpt) {
  if (nextOpt.isNone) {
    return of(null);
  }

  const [imageHash, threshold] = nextOpt.unwrap();
  return api.derive.democracy.preimage(imageHash).pipe(map(image => ({
    image,
    imageHash,
    threshold
  })));
}

export function nextExternal(instanceId, api) {
  return memo(instanceId, () => {
    var _api$query$democracy;

    return (_api$query$democracy = api.query.democracy) !== null && _api$query$democracy !== void 0 && _api$query$democracy.nextExternal ? api.query.democracy.nextExternal().pipe(switchMap(nextOpt => withImage(api, nextOpt))) : of(null);
  });
}