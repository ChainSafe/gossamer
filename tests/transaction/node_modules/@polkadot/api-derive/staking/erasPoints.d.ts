import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveEraPoints } from '../types';
export declare function _erasPoints(instanceId: string, api: DeriveApi): (eras: EraIndex[], withActive: boolean) => Observable<DeriveEraPoints[]>;
export declare const erasPoints: (instanceId: string, api: DeriveApi) => (withActive?: boolean | undefined) => Observable<DeriveEraPoints[]>;
