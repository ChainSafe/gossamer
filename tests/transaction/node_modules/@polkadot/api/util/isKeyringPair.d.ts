import type { AccountId, Address } from '@polkadot/types/interfaces';
import type { IKeyringPair } from '@polkadot/types/types';
export declare function isKeyringPair(account: string | IKeyringPair | AccountId | Address): account is IKeyringPair;
