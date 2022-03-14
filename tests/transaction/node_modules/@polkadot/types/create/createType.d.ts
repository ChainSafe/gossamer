import type { Codec, Registry } from '@polkadot/types-codec/types';
import type { DetectCodec } from '../types';
/**
 * Create an instance of a `type` with a given `params`.
 * @param type - A recognizable string representing the type to create an
 * instance from
 * @param params - The value to instantiate the type with
 */
export declare function createType<T extends Codec = Codec, K extends string = string>(registry: Registry, type: K, ...params: unknown[]): DetectCodec<T, K>;
