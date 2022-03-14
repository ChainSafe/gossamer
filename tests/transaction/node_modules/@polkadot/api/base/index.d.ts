import type { SignerPayloadRawBase } from '@polkadot/types/types';
import type { ApiOptions, ApiTypes, DecorateMethod, Signer } from '../types';
import { Getters } from './Getters';
interface KeyringSigner {
    sign(message: Uint8Array): Uint8Array;
}
interface SignerRawOptions {
    signer?: Signer;
}
export declare abstract class ApiBase<ApiType extends ApiTypes> extends Getters<ApiType> {
    /**
     * @description Create an instance of the class
     *
     * @param options Options object to create API instance or a Provider instance
     *
     * @example
     * <BR>
     *
     * ```javascript
     * import Api from '@polkadot/api/promise';
     *
     * const api = new Api().isReady();
     *
     * api.rpc.subscribeNewHeads((header) => {
     *   console.log(`new block #${header.number.toNumber()}`);
     * });
     * ```
     */
    constructor(options: ApiOptions | undefined, type: ApiTypes, decorateMethod: DecorateMethod<ApiType>);
    /**
     * @description Connect from the underlying provider, halting all network traffic
     */
    connect(): Promise<void>;
    /**
     * @description Disconnect from the underlying provider, halting all network traffic
     */
    disconnect(): Promise<void>;
    /**
     * @description Set an external signer which will be used to sign extrinsic when account passed in is not KeyringPair
     */
    setSigner(signer: Signer): void;
    /**
     * @description Signs a raw signer payload, string or Uint8Array
     */
    sign(address: KeyringSigner | string, data: SignerPayloadRawBase, { signer }?: SignerRawOptions): Promise<string>;
}
export {};
