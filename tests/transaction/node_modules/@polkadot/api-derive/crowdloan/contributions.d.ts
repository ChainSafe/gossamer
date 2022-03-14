/// <reference types="bn.js" />
import type { Observable } from 'rxjs';
import type { BN } from '@polkadot/util';
import type { DeriveApi, DeriveContributions } from '../types';
export declare function contributions(instanceId: string, api: DeriveApi): (paraId: string | number | BN) => Observable<DeriveContributions>;
