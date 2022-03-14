import type { AccountId, Digest } from '@polkadot/types/interfaces';
export declare function extractAuthor(digest: Digest, sessionValidators?: AccountId[]): AccountId | undefined;
