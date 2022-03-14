import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveEraPrefs } from '../types';
export declare function _eraPrefs(instanceId: string, api: DeriveApi): (era: EraIndex, withActive: boolean) => Observable<DeriveEraPrefs>;
export declare const eraPrefs: (instanceId: string, api: DeriveApi) => (era: EraIndex) => Observable<DeriveEraPrefs>;
export declare const _erasPrefs: (instanceId: string, api: DeriveApi) => (eras: EraIndex[], withActive: boolean) => Observable<DeriveEraPrefs[]>;
export declare const erasPrefs: (instanceId: string, api: DeriveApi) => (withActive?: boolean | undefined) => Observable<DeriveEraPrefs[]>;
