// Copyright 2017-2022 @polkadot/x-randomvalues authors & contributors
// SPDX-License-Identifier: Apache-2.0
import crypto from 'crypto';
export { packageInfo } from "./packageInfo.js";
export function getRandomValues(output) {
  const bytes = crypto.randomBytes(output.length);

  for (let i = 0; i < bytes.length; i++) {
    output[i] = bytes[i];
  }

  return output;
}