/// <reference types="bn.js" />
import type { Observable } from 'rxjs';
import type { BN } from '@polkadot/util';
import type { DeriveApi } from '../types';
export declare function childKey(instanceId: string, api: DeriveApi): (paraId: string | number | BN) => Observable<string | null>;
