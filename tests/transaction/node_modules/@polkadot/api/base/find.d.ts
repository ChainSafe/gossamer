import type { CallFunction, Registry, RegistryError } from '@polkadot/types/types';
export declare function findCall(registry: Registry, callIndex: Uint8Array | string): CallFunction;
export declare function findError(registry: Registry, errorIndex: Uint8Array | string): RegistryError;
