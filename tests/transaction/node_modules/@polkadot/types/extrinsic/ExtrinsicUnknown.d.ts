import type { Registry } from '@polkadot/types-codec/types';
import type { ExtrinsicOptions } from './types';
import { Struct } from '@polkadot/types-codec';
/**
 * @name GenericExtrinsicUnknown
 * @description
 * A default handler for extrinsics where the version is not known (default throw)
 */
export declare class GenericExtrinsicUnknown extends Struct {
    constructor(registry: Registry, value?: unknown, { isSigned, version }?: Partial<ExtrinsicOptions>);
}
