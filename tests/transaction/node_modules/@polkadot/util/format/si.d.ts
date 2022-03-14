import type { SiDef } from '../types';
export declare const SI_MID = 8;
export declare const SI: SiDef[];
export declare function findSi(type: string): SiDef;
export declare function calcSi(text: string, decimals: number, forceUnit?: string): SiDef;
