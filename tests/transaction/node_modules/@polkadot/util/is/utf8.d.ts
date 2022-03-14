/// <reference types="node" />
import type { HexString } from '../types';
/**
 * @name isUtf8
 * @summary Tests if the input is valid Utf8
 * @description
 * Checks to see if the input string or Uint8Array is valid Utf8
 */
export declare function isUtf8(value?: HexString | number[] | Buffer | Uint8Array | string | null): boolean;
