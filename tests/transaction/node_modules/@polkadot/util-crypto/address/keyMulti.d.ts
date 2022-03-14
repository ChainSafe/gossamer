/// <reference types="bn.js" />
import type { BN } from '@polkadot/util';
import type { HexString } from '@polkadot/util/types';
export declare function createKeyMulti(who: (HexString | Uint8Array | string)[], threshold: bigint | BN | number): Uint8Array;
