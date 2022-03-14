import type { Observable } from 'rxjs';
import type { HeaderExtended } from '../type/types';
import type { DeriveApi } from '../types';
/**
 * @name subscribeNewHeads
 * @returns A header with the current header (including extracted author)
 * @description An observable of the current block header and it's author
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.chain.subscribeNewHeads((header) => {
 *   console.log(`block #${header.number} was authored by ${header.author}`);
 * });
 * ```
 */
export declare function subscribeNewHeads(instanceId: string, api: DeriveApi): () => Observable<HeaderExtended>;
