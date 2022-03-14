// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { BigInt } from '@polkadot/x-bigint';
export const hasBigInt = typeof BigInt === 'function' && typeof BigInt.asIntN === 'function';
export const hasBuffer = typeof Buffer !== 'undefined';
export const hasCjs = typeof require === 'function' && typeof module !== 'undefined';
export const hasDirname = typeof __dirname !== 'undefined';
export const hasEsm = !hasCjs;
export const hasProcess = typeof process === 'object';
export const hasWasm = typeof WebAssembly !== 'undefined';