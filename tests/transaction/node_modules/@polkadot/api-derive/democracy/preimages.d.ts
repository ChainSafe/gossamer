import type { Observable } from 'rxjs';
import type { Hash } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveProposalImage } from '../types';
export declare function preimages(instanceId: string, api: DeriveApi): (hashes: (Hash | Uint8Array | string)[]) => Observable<(DeriveProposalImage | undefined)[]>;
export declare const preimage: (instanceId: string, api: DeriveApi) => (hash: string | Hash | Uint8Array) => Observable<DeriveProposalImage | undefined>;
