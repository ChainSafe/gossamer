// Copyright 2017-2022 @polkadot/keyring authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { objectSpread } from '@polkadot/util';
import { jsonEncryptFormat } from '@polkadot/util-crypto';
export function pairToJson(type, {
  address,
  meta
}, encoded, isEncrypted) {
  return objectSpread(jsonEncryptFormat(encoded, ['pkcs8', type], isEncrypted), {
    address,
    meta
  });
}