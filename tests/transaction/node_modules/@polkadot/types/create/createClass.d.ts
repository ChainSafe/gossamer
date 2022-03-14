import type { Codec, CodecClass, Registry } from '@polkadot/types-codec/types';
import type { DetectCodec } from '../types';
export declare function createClass<T extends Codec = Codec, K extends string = string>(registry: Registry, type: K): CodecClass<DetectCodec<T, K>>;
