import type { Observable } from 'rxjs';
import type { ParaId } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveParachainFull } from '../types';
export declare function info(instanceId: string, api: DeriveApi): (id: ParaId | number) => Observable<DeriveParachainFull | null>;
