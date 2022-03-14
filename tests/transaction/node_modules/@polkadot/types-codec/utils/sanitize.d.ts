interface SanitizeOptions {
    allowNamespaces?: boolean;
}
declare type Mapper = (value: string, options?: SanitizeOptions) => string;
export declare function findClosing(value: string, start: number): number;
export declare function alias(src: string, dest: string, withChecks?: boolean): Mapper;
export declare function cleanupCompact(): Mapper;
export declare function flattenSingleTuple(): Mapper;
export declare function removeExtensions(type: string, isSized: boolean): Mapper;
export declare function removeColons(): Mapper;
export declare function removeGenerics(): Mapper;
export declare function removePairOf(): Mapper;
export declare function removeTraits(): Mapper;
export declare function removeWrap(check: string): Mapper;
export declare function sanitize(value: String | string, options?: SanitizeOptions): string;
export {};
