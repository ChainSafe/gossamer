import type { SignOptions } from '@polkadot/keyring/types';
import type { Registry } from '@polkadot/types-codec/types';
import type { IKeyringPair } from '../types';
export declare function sign(registry: Registry, signerPair: IKeyringPair, u8a: Uint8Array, options?: SignOptions): Uint8Array;
