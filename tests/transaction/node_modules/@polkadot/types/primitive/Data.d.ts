import type { Registry } from '@polkadot/types-codec/types';
import type { H256 } from '../interfaces/runtime';
import { Bytes, Enum } from '@polkadot/types-codec';
/**
 * @name Data
 * @description
 * A [[Data]] container with node, raw or hashed data
 */
export declare class Data extends Enum {
    constructor(registry: Registry, value?: Record<string, any> | Uint8Array | Enum | string);
    get asBlakeTwo256(): H256;
    get asKeccak256(): H256;
    get asRaw(): Bytes;
    get asSha256(): H256;
    get asShaThree256(): H256;
    get isBlakeTwo256(): boolean;
    get isKeccak256(): boolean;
    get isNone(): boolean;
    get isRaw(): boolean;
    get isSha256(): boolean;
    get isShaThree256(): boolean;
    /**
     * @description The encoded length
     */
    get encodedLength(): number;
    /**
     * @description Encodes the value as a Uint8Array as per the SCALE specifications
     */
    toU8a(): Uint8Array;
}
