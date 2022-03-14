import type { Vec } from '@polkadot/types-codec';
import type { AnyNumber, AnyU8a, IU8a, Registry } from '@polkadot/types-codec/types';
import type { GenericExtrinsic } from '../extrinsic/Extrinsic';
import type { Digest, DigestItem, Header } from '../interfaces/runtime';
import { Struct } from '@polkadot/types-codec';
export interface HeaderValue {
    digest?: Digest | {
        logs: DigestItem[] | string[];
    };
    extrinsicsRoot?: AnyU8a;
    number?: AnyNumber;
    parentHash?: AnyU8a;
    stateRoot?: AnyU8a;
}
export interface BlockValue {
    extrinsics?: AnyU8a[];
    header?: HeaderValue;
}
/**
 * @name GenericBlock
 * @description
 * A block encoded with header and extrinsics
 */
export declare class GenericBlock extends Struct {
    constructor(registry: Registry, value?: BlockValue | Uint8Array);
    /**
     * @description Encodes a content [[Hash]] for the block
     */
    get contentHash(): IU8a;
    /**
     * @description The [[Extrinsic]] contained in the block
     */
    get extrinsics(): Vec<GenericExtrinsic>;
    /**
     * @description Block/header [[Hash]]
     */
    get hash(): IU8a;
    /**
     * @description The [[Header]] of the block
     */
    get header(): Header;
}
