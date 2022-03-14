/// <reference types="bn.js" />
import type { Observable } from 'rxjs';
import type { BN } from '@polkadot/util';
import type { DeriveApi, DeriveOwnContributions } from '../types';
export declare function ownContributions(instanceId: string, api: DeriveApi): (paraId: string | number | BN, keys: string[]) => Observable<DeriveOwnContributions>;
