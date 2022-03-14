/// <reference types="bn.js" />
import type { BN } from '../bn/bn';
import type { SiDef, ToBn } from '../types';
interface Defaults {
    decimals: number;
    unit: string;
}
interface SetDefaults {
    decimals?: number[] | number;
    unit?: string[] | string;
}
interface Options {
    decimals?: number;
    forceUnit?: string;
    withSi?: boolean;
    withSiFull?: boolean;
    withUnit?: boolean | string;
}
interface BalanceFormatter {
    <ExtToBn extends ToBn>(input?: number | string | BN | bigint | ExtToBn, options?: Options, decimals?: number): string;
    calcSi(text: string, decimals?: number): SiDef;
    findSi(type: string): SiDef;
    getDefaults(): Defaults;
    getOptions(decimals?: number): SiDef[];
    setDefaults(defaults: SetDefaults): void;
}
export declare const formatBalance: BalanceFormatter;
export {};
