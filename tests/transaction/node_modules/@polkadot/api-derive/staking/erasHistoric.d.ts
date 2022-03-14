import type { Observable } from 'rxjs';
import type { EraIndex } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
export declare function erasHistoric(instanceId: string, api: DeriveApi): (withActive?: boolean) => Observable<EraIndex[]>;
