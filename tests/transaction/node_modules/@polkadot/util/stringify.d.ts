/**
 * @name stringify
 * @summary Performs a JSON.stringify, with BigInt handling
 * @description A wrapper for JSON.stringify that handles BigInt values transparently, converting them to string. No differences from the native JSON.stringify function otherwise.
 */
export declare function stringify(value: unknown, space?: string | number): string;
