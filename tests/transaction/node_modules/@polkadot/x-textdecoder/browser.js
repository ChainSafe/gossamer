// Copyright 2017-2022 @polkadot/x-textencoder authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { extractGlobal } from '@polkadot/x-global';
import { TextDecoder as Fallback } from "./fallback.js";
export { packageInfo } from "./packageInfo.js";
export const TextDecoder = extractGlobal('TextDecoder', Fallback);