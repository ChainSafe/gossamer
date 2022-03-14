import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveEraSlashes } from '../types';
export declare function _eraSlashes(instanceId: string, api: DeriveApi): (era: EraIndex, withActive: boolean) => Observable<DeriveEraSlashes>;
export declare const eraSlashes: (instanceId: string, api: DeriveApi) => (era: EraIndex) => Observable<DeriveEraSlashes>;
export declare const _erasSlashes: (instanceId: string, api: DeriveApi) => (eras: EraIndex[], withActive: boolean) => Observable<DeriveEraSlashes[]>;
export declare const erasSlashes: (instanceId: string, api: DeriveApi) => (withActive?: boolean | undefined) => Observable<DeriveEraSlashes[]>;
