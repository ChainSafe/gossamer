// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import ed2curve from 'ed2curve';
import { assertReturn } from '@polkadot/util';
export function convertSecretKeyToCurve25519(secretKey) {
  return ed2curve.convertSecretKey(secretKey);
}
export function convertPublicKeyToCurve25519(publicKey) {
  return assertReturn(ed2curve.convertPublicKey(publicKey), 'Unable to convert publicKey to ed25519');
}