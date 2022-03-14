import type { Bytes, Option, u32, Vec } from '@polkadot/types';
import type { CollatorId, ParaId, ParaInfo, Retriable, UpwardMessage } from '@polkadot/types/interfaces';
import type { ITuple } from '@polkadot/types/types';
export declare type ParaInfoResult = Option<ParaInfo>;
export declare type PendingSwap = Option<ParaId>;
export declare type Active = Vec<ITuple<[ParaId, Option<ITuple<[CollatorId, Retriable]>>]>>;
export declare type RetryQueue = Vec<Vec<ITuple<[ParaId, CollatorId]>>>;
export declare type SelectedThreads = Vec<Vec<ITuple<[ParaId, CollatorId]>>>;
export declare type Code = Bytes;
export declare type Heads = Bytes;
export declare type RelayDispatchQueue = Vec<UpwardMessage>;
export declare type RelayDispatchQueueSize = ITuple<[u32, u32]>;
export declare type DidUpdate = Option<Vec<ParaId>>;
export interface DeriveParachainActive {
    collatorId: CollatorId;
    isRetriable: boolean;
    retries: number;
}
export interface DeriveParachainInfo extends ParaInfo {
    id: ParaId;
    icon?: string;
    name?: string;
    owner?: string;
}
export interface DeriveParachain {
    didUpdate: boolean;
    pendingSwapId: ParaId | null;
    id: ParaId;
    info: DeriveParachainInfo | null;
    relayDispatchQueueSize?: number;
}
export interface DeriveParachainFull extends DeriveParachain {
    active: DeriveParachainActive | null;
    heads: Bytes | null;
    relayDispatchQueue: UpwardMessage[];
    retryCollators: (CollatorId | null)[];
    selectedCollators: (CollatorId | null)[];
}
