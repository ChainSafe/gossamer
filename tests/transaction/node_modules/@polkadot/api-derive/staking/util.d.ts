import type { Observable } from 'rxjs';
import type { ObsInnerType } from '@polkadot/api-base/types';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { ExactDerive } from '../derive';
import type { DeriveApi } from '../types';
declare type ApplyReturn<T extends keyof ExactDerive['staking']> = ReturnType<ExactDerive['staking'][T]>;
export declare function filterEras<T extends {
    era: EraIndex;
}>(eras: EraIndex[], list: T[]): EraIndex[];
export declare function erasHistoricApply<F extends '_erasExposure' | '_erasPoints' | '_erasPrefs' | '_erasRewards' | '_erasSlashes'>(fn: F): (instanceId: string, api: DeriveApi) => (withActive?: boolean) => ApplyReturn<F>;
export declare function erasHistoricApplyAccount<F extends '_ownExposures' | '_ownSlashes' | '_stakerPoints' | '_stakerPrefs' | '_stakerSlashes'>(fn: F): (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, withActive?: boolean) => ApplyReturn<F>;
export declare function singleEra<F extends '_eraExposure' | '_eraPrefs' | '_eraSlashes'>(fn: F): (instanceId: string, api: DeriveApi) => (era: EraIndex) => ApplyReturn<F>;
export declare function combineEras<F extends '_eraExposure' | '_eraPrefs' | '_eraSlashes'>(fn: F): (instanceId: string, api: DeriveApi) => (eras: EraIndex[], withActive: boolean) => Observable<ObsInnerType<ApplyReturn<F>>[]>;
export {};
