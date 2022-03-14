import type { ProviderInterface } from '@polkadot/rpc-provider/types';
import type { AnyNumber, DefinitionRpc, DefinitionRpcExt, DefinitionRpcSub, Registry } from '@polkadot/types/types';
export { packageInfo } from './packageInfo';
export * from './util';
/**
 * @name Rpc
 * @summary The API may use a HTTP or WebSockets provider.
 * @description It allows for querying a Polkadot Client Node.
 * WebSockets provider is recommended since HTTP provider only supports basic querying.
 *
 * ```mermaid
 * graph LR;
 *   A[Api] --> |WebSockets| B[WsProvider];
 *   B --> |endpoint| C[ws://127.0.0.1:9944]
 * ```
 *
 * @example
 * <BR>
 *
 * ```javascript
 * import Rpc from '@polkadot/rpc-core';
 * import { WsProvider } from '@polkadot/rpc-provider/ws';
 *
 * const provider = new WsProvider('ws://127.0.0.1:9944');
 * const rpc = new Rpc(provider);
 * ```
 */
export declare class RpcCore {
    #private;
    readonly mapping: Map<string, DefinitionRpcExt>;
    readonly provider: ProviderInterface;
    readonly sections: string[];
    /**
     * @constructor
     * Default constructor for the Api Object
     * @param  {ProviderInterface} provider An API provider using HTTP or WebSocket
     */
    constructor(instanceId: string, registry: Registry, provider: ProviderInterface, userRpc?: Record<string, Record<string, DefinitionRpc | DefinitionRpcSub>>);
    /**
     * @description Returns the connected status of a provider
     */
    get isConnected(): boolean;
    /**
     * @description Manually connect from the attached provider
     */
    connect(): Promise<void>;
    /**
     * @description Manually disconnect from the attached provider
     */
    disconnect(): Promise<void>;
    /**
     * @description Sets a registry swap (typically from Api)
     */
    setRegistrySwap(registrySwap: (blockHash: Uint8Array) => Promise<{
        registry: Registry;
    }>): void;
    /**
     * @description Sets a function to resolve block hash from block number
     */
    setResolveBlockHash(resolveBlockHash: (blockNumber: AnyNumber) => Promise<Uint8Array>): void;
    addUserInterfaces(userRpc: Record<string, Record<string, DefinitionRpc | DefinitionRpcSub>>): void;
    private _memomize;
    private _formatResult;
    private _createMethodSend;
    private _createSubscriber;
    private _createMethodSubscribe;
    private _formatInputs;
    private _formatOutput;
    private _formatStorageData;
    private _formatStorageSet;
    private _formatStorageSetEntry;
    private _newType;
}
