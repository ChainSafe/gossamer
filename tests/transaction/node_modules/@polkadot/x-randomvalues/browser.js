// Copyright 2017-2022 @polkadot/x-randomvalues authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { xglobal } from '@polkadot/x-global';
export { packageInfo } from "./packageInfo.js";
export function getRandomValues(arr) {
  // We use x-global here - this prevents packagers such as rollup
  // confusing this with the "normal" Node.js import and stubbing it
  // (and also aligns with eg. x-fetch, where x-global is used)
  return xglobal.crypto.getRandomValues(arr);
}