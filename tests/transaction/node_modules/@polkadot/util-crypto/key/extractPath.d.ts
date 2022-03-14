import { DeriveJunction } from './DeriveJunction';
export interface ExtractResult {
    parts: null | string[];
    path: DeriveJunction[];
}
/**
 * @description Extract derivation junctions from the supplied path
 */
export declare function keyExtractPath(derivePath: string): ExtractResult;
