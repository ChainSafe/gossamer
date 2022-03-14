import type { U8aLike } from '../types';
/**
 * @name isAscii
 * @summary Tests if the input is printable ASCII
 * @description
 * Checks to see if the input string or Uint8Array is printable ASCII, 32-127 + formatters
 */
export declare function isAscii(value?: U8aLike | null): boolean;
