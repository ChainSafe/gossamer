import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveEraExposure } from '../types';
export declare function _eraExposure(instanceId: string, api: DeriveApi): (era: EraIndex, withActive?: boolean) => Observable<DeriveEraExposure>;
export declare const eraExposure: (instanceId: string, api: DeriveApi) => (era: EraIndex) => Observable<DeriveEraExposure>;
export declare const _erasExposure: (instanceId: string, api: DeriveApi) => (eras: EraIndex[], withActive: boolean) => Observable<DeriveEraExposure[]>;
export declare const erasExposure: (instanceId: string, api: DeriveApi) => (withActive?: boolean | undefined) => Observable<DeriveEraExposure[]>;
