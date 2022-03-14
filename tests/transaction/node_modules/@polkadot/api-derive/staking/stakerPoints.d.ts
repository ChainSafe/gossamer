import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveStakerPoints } from '../types';
export declare function _stakerPoints(instanceId: string, api: DeriveApi): (accountId: Uint8Array | string, eras: EraIndex[], withActive: boolean) => Observable<DeriveStakerPoints[]>;
export declare const stakerPoints: (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, withActive?: boolean | undefined) => Observable<DeriveStakerPoints[]>;
