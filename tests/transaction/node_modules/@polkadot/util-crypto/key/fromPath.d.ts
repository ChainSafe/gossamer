import type { Keypair, KeypairType } from '../types';
import { DeriveJunction } from './DeriveJunction';
export declare function keyFromPath(pair: Keypair, path: DeriveJunction[], type: KeypairType): Keypair;
