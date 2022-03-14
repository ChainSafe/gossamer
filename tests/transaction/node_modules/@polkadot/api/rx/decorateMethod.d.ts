import type { Codec } from '@polkadot/types/types';
import type { DecorateFn } from '../types';
export declare function toRxMethod<M extends DecorateFn<Codec>>(method: M): M;
