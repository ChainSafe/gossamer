import type { EraIndex } from '@polkadot/types/interfaces';
export declare function getEraCache<T>(CACHE_KEY: string, era: EraIndex, withActive?: boolean): [string, T | undefined];
export declare function getEraMultiCache<T>(CACHE_KEY: string, eras: EraIndex[], withActive?: boolean): T[];
export declare function setEraCache<T extends {
    era: EraIndex;
}>(cacheKey: string, withActive: boolean, value: T): T;
export declare function setEraMultiCache<T extends {
    era: EraIndex;
}>(CACHE_KEY: string, withActive: boolean, values: T[]): T[];
export declare function filterCachedEras<T extends {
    era: EraIndex;
}>(eras: EraIndex[], cached: T[], query: T[]): T[];
