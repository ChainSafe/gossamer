/// <reference types="bn.js" />
import type { BN } from '../bn/bn';
import type { ToBn } from '../types';
export declare function formatElapsed<ExtToBn extends ToBn>(now?: Date | null, value?: bigint | BN | ExtToBn | Date | number | null): string;
