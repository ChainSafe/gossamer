/// <reference types="bn.js" />
import type { BN } from '../bn/bn';
import type { ToBn } from '../types';
export declare function formatNumber<ExtToBn extends ToBn>(value?: ExtToBn | BN | bigint | number | null): string;
