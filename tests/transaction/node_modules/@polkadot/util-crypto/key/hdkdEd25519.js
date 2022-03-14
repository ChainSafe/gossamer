// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { ed25519DeriveHard, ed25519PairFromSeed } from "../ed25519/index.js";
import { createSeedDeriveFn } from "./hdkdDerive.js";
export const keyHdkdEd25519 = createSeedDeriveFn(ed25519PairFromSeed, ed25519DeriveHard);