/// <reference types="bn.js" />
import type { FrameSystemEventRecord } from '@polkadot/types/lookup';
import type { Vec } from '@polkadot/types-codec';
import type { BN } from '@polkadot/util';
interface Changes {
    added: string[];
    blockHash: string;
    removed: string[];
}
export declare function extractContributed(paraId: string | number | BN, events: Vec<FrameSystemEventRecord>): Changes;
export {};
