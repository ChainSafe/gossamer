// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
export function getSiName(lookup, type) {
  const typeDef = lookup.getTypeDef(type);
  return typeDef.lookupName || typeDef.type;
}