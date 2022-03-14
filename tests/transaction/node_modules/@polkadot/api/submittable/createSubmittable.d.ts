import type { Call } from '@polkadot/types/interfaces';
import type { ApiInterfaceRx, ApiTypes } from '../types';
import type { SubmittableExtrinsic } from './types';
import { ApiBase } from '../base';
declare type Creator<ApiType extends ApiTypes> = (extrinsic: Call | Uint8Array | string) => SubmittableExtrinsic<ApiType>;
export declare function createSubmittable<ApiType extends ApiTypes>(apiType: ApiTypes, api: ApiInterfaceRx, decorateMethod: ApiBase<ApiType>['_decorateMethod']): Creator<ApiType>;
export {};
