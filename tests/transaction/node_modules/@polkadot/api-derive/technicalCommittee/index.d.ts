export declare const members: (instanceId: string, api: import("../types").DeriveApi) => () => import("rxjs").Observable<import("@polkadot/types/interfaces/runtime/types").AccountId[]>;
export declare const hasProposals: (instanceId: string, api: import("../types").DeriveApi) => () => import("rxjs").Observable<boolean>;
export declare const proposal: (instanceId: string, api: import("../types").DeriveApi) => (hash: string | import("@polkadot/types/interfaces/runtime/types").Hash | Uint8Array) => import("rxjs").Observable<import("../types").DeriveCollectiveProposal | null>;
export declare const proposalCount: (instanceId: string, api: import("../types").DeriveApi) => () => import("rxjs").Observable<import("@polkadot/types-codec/primitive/U32").u32 | null>;
export declare const proposalHashes: (instanceId: string, api: import("../types").DeriveApi) => () => import("rxjs").Observable<import("@polkadot/types/interfaces/runtime/types").Hash[]>;
export declare const proposals: (instanceId: string, api: import("../types").DeriveApi) => () => import("rxjs").Observable<import("../types").DeriveCollectiveProposal[]>;
export declare const prime: (instanceId: string, api: import("../types").DeriveApi) => () => import("rxjs").Observable<import("@polkadot/types/interfaces/runtime/types").AccountId | null>;
