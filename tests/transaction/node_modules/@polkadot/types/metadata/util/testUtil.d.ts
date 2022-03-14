import type { Registry } from '@polkadot/types-codec/types';
import type { Check } from './types';
/** @internal */
export declare function decodeLatestMeta(registry: Registry, type: string, version: number, { compare, data, types }: Check): void;
/** @internal */
export declare function toLatest(registry: Registry, version: number, { data }: Check, withThrow?: boolean): void;
/** @internal */
export declare function defaultValues(registry: Registry, { data, fails }: Check, withThrow?: boolean, withFallbackCheck?: boolean): void;
export declare function testMeta(version: number, matchers: Record<string, Check>, withFallback?: boolean): void;
