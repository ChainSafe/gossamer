import type { Text, u8 } from '@polkadot/types-codec';
import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataLatest, SiField, SiVariant } from '../../../interfaces';
import type { PortableRegistry } from '../../../metadata';
import type { Errors } from '../types';
interface ItemMeta {
    args: string[];
    name: Text;
    fields: SiField[];
    index: u8;
    docs: Text[];
}
export declare function variantToMeta(lookup: PortableRegistry, variant: SiVariant): ItemMeta;
/** @internal */
export declare function decorateErrors(registry: Registry, { lookup, pallets }: MetadataLatest, version: number): Errors;
export {};
