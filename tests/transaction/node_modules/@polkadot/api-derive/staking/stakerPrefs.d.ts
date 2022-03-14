import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveStakerPrefs } from '../types';
export declare function _stakerPrefs(instanceId: string, api: DeriveApi): (accountId: Uint8Array | string, eras: EraIndex[], withActive: boolean) => Observable<DeriveStakerPrefs[]>;
export declare const stakerPrefs: (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, withActive?: boolean | undefined) => Observable<DeriveStakerPrefs[]>;
