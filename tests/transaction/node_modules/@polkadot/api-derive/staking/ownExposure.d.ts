import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveOwnExposure } from '../types';
export declare function _ownExposures(instanceId: string, api: DeriveApi): (accountId: Uint8Array | string, eras: EraIndex[], withActive: boolean) => Observable<DeriveOwnExposure[]>;
export declare const ownExposure: (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, era: EraIndex) => Observable<DeriveOwnExposure>;
export declare const ownExposures: (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, withActive?: boolean | undefined) => Observable<DeriveOwnExposure[]>;
