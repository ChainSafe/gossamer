import type { AccountId } from '@polkadot/types/interfaces';
export type { AccountId } from '@polkadot/types/interfaces';
export declare const members: (section: import("./types").Collective) => (instanceId: string, api: import("../types").DeriveApi) => () => import("rxjs").Observable<AccountId[]>;
