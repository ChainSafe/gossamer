import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveEraRewards } from '../types';
export declare function _erasRewards(instanceId: string, api: DeriveApi): (eras: EraIndex[], withActive: boolean) => Observable<DeriveEraRewards[]>;
export declare const erasRewards: (instanceId: string, api: DeriveApi) => (withActive?: boolean | undefined) => Observable<DeriveEraRewards[]>;
