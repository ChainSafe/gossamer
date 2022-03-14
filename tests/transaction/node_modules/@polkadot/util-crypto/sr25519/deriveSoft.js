// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { sr25519DeriveKeypairSoft } from '@polkadot/wasm-crypto';
import { createDeriveFn } from "./derive.js";
export const sr25519DeriveSoft = createDeriveFn(sr25519DeriveKeypairSoft);