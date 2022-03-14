import type { Observable } from 'rxjs';
import type { EventRecord, Hash, SignedBlock } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
interface Result {
    block: SignedBlock;
    events: EventRecord[];
}
export declare function events(instanceId: string, api: DeriveApi): (at: Hash) => Observable<Result>;
export {};
