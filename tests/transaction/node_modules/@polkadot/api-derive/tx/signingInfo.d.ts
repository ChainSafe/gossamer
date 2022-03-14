import type { Observable } from 'rxjs';
import type { Header, Index } from '@polkadot/types/interfaces';
import type { AnyNumber, Codec, IExtrinsicEra } from '@polkadot/types/types';
import type { DeriveApi } from '../types';
interface Result {
    header: Header | null;
    mortalLength: number;
    nonce: Index;
}
export declare function signingInfo(_instanceId: string, api: DeriveApi): (address: string, nonce?: AnyNumber | Codec, era?: IExtrinsicEra | number) => Observable<Result>;
export {};
