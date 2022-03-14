import { DeriveJunction } from './DeriveJunction';
export interface ExtractResult {
    derivePath: string;
    password?: string;
    path: DeriveJunction[];
    phrase: string;
}
/**
 * @description Extracts the phrase, path and password from a SURI format for specifying secret keys `<secret>/<soft-key>//<hard-key>///<password>` (the `///password` may be omitted, and `/<soft-key>` and `//<hard-key>` maybe repeated and mixed).
 */
export declare function keyExtractSuri(suri: string): ExtractResult;
