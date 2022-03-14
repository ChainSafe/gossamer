import type { Codec, Registry } from '@polkadot/types-codec/types';
import type { CreateOptions } from '../types';
export declare function createTypeUnsafe<T extends Codec = Codec, K extends string = string>(registry: Registry, type: K, params?: unknown[], options?: CreateOptions): T;
