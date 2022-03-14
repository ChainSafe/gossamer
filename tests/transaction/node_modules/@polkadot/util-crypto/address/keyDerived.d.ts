/// <reference types="bn.js" />
import type { BN } from '@polkadot/util';
import type { HexString } from '@polkadot/util/types';
export declare function createKeyDerived(who: HexString | Uint8Array | string, index: bigint | BN | number): Uint8Array;
