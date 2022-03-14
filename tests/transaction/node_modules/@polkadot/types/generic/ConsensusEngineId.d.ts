import type { AnyU8a, Registry } from '@polkadot/types-codec/types';
import type { AccountId } from '../interfaces';
import { Bytes, U8aFixed } from '@polkadot/types-codec';
export declare const CID_AURA: Uint8Array;
export declare const CID_BABE: Uint8Array;
export declare const CID_GRPA: Uint8Array;
export declare const CID_POW: Uint8Array;
/**
 * @name GenericConsensusEngineId
 * @description
 * A 4-byte identifier identifying the engine
 */
export declare class GenericConsensusEngineId extends U8aFixed {
    constructor(registry: Registry, value?: AnyU8a);
    /**
     * @description `true` if the engine matches aura
     */
    get isAura(): boolean;
    /**
     * @description `true` is the engine matches babe
     */
    get isBabe(): boolean;
    /**
     * @description `true` is the engine matches grandpa
     */
    get isGrandpa(): boolean;
    /**
     * @description `true` is the engine matches pow
     */
    get isPow(): boolean;
    /**
     * @description From the input bytes, decode into an author
     */
    extractAuthor(bytes: Bytes, sessionValidators: AccountId[]): AccountId | undefined;
    /**
     * @description Converts the Object to to a human-friendly JSON, with additional fields, expansion and formatting of information
     */
    toHuman(): string;
    /**
     * @description Returns the base runtime type name for this instance
     */
    toRawType(): string;
    /**
     * @description Override the default toString to return a 4-byte string
     */
    toString(): string;
}
