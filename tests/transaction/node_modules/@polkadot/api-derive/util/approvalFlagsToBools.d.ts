import type { Vec } from '@polkadot/types';
import type { ApprovalFlag } from '@polkadot/types/interfaces/elections';
/** @internal */
export declare function approvalFlagsToBools(flags: Vec<ApprovalFlag> | ApprovalFlag[]): boolean[];
