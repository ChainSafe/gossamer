import type { OverrideModuleType, Registry } from '../types';
/**
 * @description Get types for specific modules (metadata override)
 */
export declare function getAliasTypes({ knownTypes }: Registry, section: string): OverrideModuleType;
