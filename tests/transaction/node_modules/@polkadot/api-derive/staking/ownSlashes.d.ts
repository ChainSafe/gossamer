import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveStakerSlashes } from '../types';
export declare function _ownSlashes(instanceId: string, api: DeriveApi): (accountId: Uint8Array | string, eras: EraIndex[], withActive: boolean) => Observable<DeriveStakerSlashes[]>;
export declare const ownSlash: (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, era: EraIndex) => Observable<DeriveStakerSlashes>;
export declare const ownSlashes: (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, withActive?: boolean | undefined) => Observable<DeriveStakerSlashes[]>;
