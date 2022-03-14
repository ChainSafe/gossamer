import type { Keypair } from '../types';
import { DeriveJunction } from './DeriveJunction';
export declare function createSeedDeriveFn(fromSeed: (seed: Uint8Array) => Keypair, derive: (seed: Uint8Array, chainCode: Uint8Array) => Uint8Array): (keypair: Keypair, junction: DeriveJunction) => Keypair;
