// Copyright 2017-2021 @polkadot/wasm-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { detectPackage } from '@polkadot/util';
import { packageInfo as asmInfo } from '@polkadot/wasm-crypto-asmjs/packageInfo';
import { packageInfo as wasmInfo } from '@polkadot/wasm-crypto-wasm/packageInfo';
import { packageInfo } from "./packageInfo.js";
detectPackage(packageInfo, typeof __dirname !== 'undefined' && __dirname, [asmInfo, wasmInfo]);