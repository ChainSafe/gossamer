import type { EventRecord, ExtrinsicStatus, H256, SignedBlock } from '@polkadot/types/interfaces';
export declare function filterEvents(txHash: H256, { block: { extrinsics, header } }: SignedBlock, allEvents: EventRecord[], status: ExtrinsicStatus): {
    events?: EventRecord[];
    txIndex?: number;
};
