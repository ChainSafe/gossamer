import type { Observable } from 'rxjs';
import type { DeriveApi } from '../types';
export declare function firstObservable<T>(obs: Observable<T[]>): Observable<T>;
export declare function firstMemo<T, A extends any[]>(fn: (api: DeriveApi, ...args: A) => Observable<T[]>): (instanceId: string, api: DeriveApi) => (...args: A) => Observable<T>;
