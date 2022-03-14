import type { SiLookupTypeId, SiVariant } from '../interfaces';
import type { PortableRegistry } from '../metadata';
interface TypeHolder {
    type: SiLookupTypeId;
}
export declare function lazyVariants<T>(lookup: PortableRegistry, { type }: TypeHolder, getName: (v: SiVariant) => string, creator: (v: SiVariant) => T): Record<string, T>;
export {};
