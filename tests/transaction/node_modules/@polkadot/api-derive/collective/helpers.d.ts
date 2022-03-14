import type { Observable } from 'rxjs';
import type { DeriveApi } from '../types';
import type { Collective } from './types';
export declare function getInstance(api: DeriveApi, section: string): DeriveApi['query']['council'];
export declare function withSection<T, F extends (...args: any[]) => Observable<T>>(section: Collective, fn: (query: DeriveApi['query']['council'], api: DeriveApi, instanceId: string) => F): (instanceId: string, api: DeriveApi) => F;
export declare function callMethod<T>(method: 'members' | 'proposals' | 'proposalCount', empty: T): (section: Collective) => (instanceId: string, api: DeriveApi) => () => Observable<T>;
