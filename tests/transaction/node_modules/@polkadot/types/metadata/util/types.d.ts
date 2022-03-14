import type { HexString } from '@polkadot/util/types';
export interface Check {
    compare: Record<string, unknown>;
    data: HexString;
    fails?: string[];
    types?: unknown[];
}
