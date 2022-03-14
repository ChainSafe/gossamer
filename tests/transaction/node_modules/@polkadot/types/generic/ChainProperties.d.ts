import type { Registry } from '@polkadot/types-codec/types';
import { Json, Option, Text, u32, Vec } from '@polkadot/types-codec';
export declare class GenericChainProperties extends Json {
    constructor(registry: Registry, value?: Map<string, unknown> | Record<string, unknown> | null);
    /**
     * @description The chain ss58Format
     */
    get ss58Format(): Option<u32>;
    /**
     * @description The decimals for each of the tokens
     */
    get tokenDecimals(): Option<Vec<u32>>;
    /**
     * @description The symbols for the tokens
     */
    get tokenSymbol(): Option<Vec<Text>>;
}
