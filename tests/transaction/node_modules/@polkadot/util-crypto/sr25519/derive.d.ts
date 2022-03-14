import type { Keypair } from '../types';
export declare function createDeriveFn(derive: (pair: Uint8Array, cc: Uint8Array) => Uint8Array): (keypair: Keypair, chainCode: Uint8Array) => Keypair;
