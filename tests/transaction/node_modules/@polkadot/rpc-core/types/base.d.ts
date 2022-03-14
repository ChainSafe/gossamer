import type { Observable } from 'rxjs';
import type { AnyFunction, Codec, DefinitionRpc } from '@polkadot/types/types';
export interface RpcInterfaceMethod {
    <T extends Codec>(...params: unknown[]): Observable<T>;
    raw(...params: unknown[]): Observable<unknown>;
    meta: DefinitionRpc;
}
export declare type AugmentedRpc<F extends AnyFunction> = F & {
    raw: <T>(...params: Parameters<F>) => Observable<T>;
    meta: DefinitionRpc;
};
