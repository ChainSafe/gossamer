import type { Vec } from '@polkadot/types-codec';
import type { Codec } from '../types';
export interface MetadataInterface<Modules extends Codec> extends Codec {
    pallets: Vec<Modules>;
}
