/**
 * @name objectProperty
 * @summary Assign a get property on the input object
 */
export declare function objectProperty(that: object, key: string, getter: (k: string) => unknown): void;
/**
 * @name objectProperties
 * @summary Assign get properties on the input object
 */
export declare function objectProperties(that: object, keys: string[], getter: (k: string, i: number) => unknown): void;
