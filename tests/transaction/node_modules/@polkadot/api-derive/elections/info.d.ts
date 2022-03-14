import type { Observable } from 'rxjs';
import type { DeriveApi } from '../types';
import type { DeriveElectionsInfo } from './types';
/**
 * @name info
 * @returns An object containing the combined results of the storage queries for
 * all relevant election module properties.
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.elections.info(({ members, candidates }) => {
 *   console.log(`There are currently ${members.length} council members and ${candidates.length} prospective council candidates.`);
 * });
 * ```
 */
export declare function info(instanceId: string, api: DeriveApi): () => Observable<DeriveElectionsInfo>;
