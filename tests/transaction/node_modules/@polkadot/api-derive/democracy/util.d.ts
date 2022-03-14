/// <reference types="bn.js" />
import type { ReferendumInfoTo239 } from '@polkadot/types/interfaces';
import type { PalletDemocracyReferendumInfo, PalletDemocracyReferendumStatus, PalletDemocracyVoteThreshold } from '@polkadot/types/lookup';
import type { Option } from '@polkadot/types-codec';
import type { DeriveReferendum, DeriveReferendumVote, DeriveReferendumVotes } from '../types';
import { BN } from '@polkadot/util';
interface ApproxState {
    votedAye: BN;
    votedNay: BN;
    votedTotal: BN;
}
export declare function compareRationals(n1: BN, d1: BN, n2: BN, d2: BN): boolean;
export declare function calcPassing(threshold: PalletDemocracyVoteThreshold, sqrtElectorate: BN, state: ApproxState): boolean;
export declare function calcVotes(sqrtElectorate: BN, referendum: DeriveReferendum, votes: DeriveReferendumVote[]): DeriveReferendumVotes;
export declare function getStatus(info: Option<PalletDemocracyReferendumInfo | ReferendumInfoTo239>): PalletDemocracyReferendumStatus | ReferendumInfoTo239 | null;
export {};
