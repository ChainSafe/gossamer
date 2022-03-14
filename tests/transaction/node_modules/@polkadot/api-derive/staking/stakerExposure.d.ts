import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
import type { DeriveStakerExposure } from './types';
export declare function _stakerExposures(instanceId: string, api: DeriveApi): (accountIds: (Uint8Array | string)[], eras: EraIndex[], withActive?: boolean) => Observable<DeriveStakerExposure[][]>;
export declare function stakerExposures(instanceId: string, api: DeriveApi): (accountIds: (Uint8Array | string)[], withActive?: boolean) => Observable<DeriveStakerExposure[][]>;
export declare const stakerExposure: (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, withActive?: boolean | undefined) => Observable<DeriveStakerExposure[]>;
