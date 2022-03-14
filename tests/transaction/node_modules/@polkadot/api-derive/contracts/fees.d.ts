import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveContractFees } from '../types';
/**
 * @name fees
 * @returns An object containing the combined results of the queries for
 * all relevant contract fees as declared in the substrate chain spec.
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.contracts.fees(([creationFee, transferFee]) => {
 *   console.log(`The fee for creating a new contract on this chain is ${creationFee} units. The fee required to call this contract is ${transferFee} units.`);
 * });
 * ```
 */
export declare function fees(instanceId: string, api: DeriveApi): () => Observable<DeriveContractFees>;
