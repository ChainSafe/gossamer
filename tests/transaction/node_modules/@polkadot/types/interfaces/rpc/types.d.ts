import type { Struct, Text, Vec, u32 } from '@polkadot/types-codec';
/** @name RpcMethods */
export interface RpcMethods extends Struct {
    readonly version: u32;
    readonly methods: Vec<Text>;
}
export declare type PHANTOM_RPC = 'rpc';
