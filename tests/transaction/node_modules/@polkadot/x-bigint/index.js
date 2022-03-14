// Copyright 2017-2022 @polkadot/x-bigint authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { xglobal } from '@polkadot/x-global';
export { packageInfo } from "./packageInfo.js";
export const BigInt = typeof xglobal.BigInt === 'function' && typeof xglobal.BigInt.asIntN === 'function' ? xglobal.BigInt : () => Number.NaN;