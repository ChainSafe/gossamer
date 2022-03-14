/**
 * @name objectKeys
 * @summary A version of Object.keys that is typed for TS
 */
export declare function objectKeys<T extends object, K extends Extract<keyof T, string>>(value: T): K[];
