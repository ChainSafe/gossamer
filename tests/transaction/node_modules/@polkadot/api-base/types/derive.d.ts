import type { Observable } from 'rxjs';
declare type DeriveCreator = (instanceId: string, api: unknown) => (...args: unknown[]) => Observable<any>;
export declare type DeriveCustom = Record<string, Record<string, DeriveCreator>>;
export {};
