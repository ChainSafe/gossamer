/**
 * @name objectValues
 * @summary A version of Object.values that is typed for TS
 */
export declare function objectValues<T extends object>(obj: T): T[keyof T][];
