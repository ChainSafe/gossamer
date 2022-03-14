import type { Logger } from '@polkadot/util/types';
import type { RpcCoder } from '../coder';
export interface HttpState {
    coder: RpcCoder;
    endpoint: string;
    l: Logger;
}
