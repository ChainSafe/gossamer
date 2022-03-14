// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { secp256k1DeriveHard } from "../secp256k1/deriveHard.js";
import { secp256k1PairFromSeed } from "../secp256k1/pair/fromSeed.js";
import { createSeedDeriveFn } from "./hdkdDerive.js";
export const keyHdkdEcdsa = createSeedDeriveFn(secp256k1PairFromSeed, secp256k1DeriveHard);