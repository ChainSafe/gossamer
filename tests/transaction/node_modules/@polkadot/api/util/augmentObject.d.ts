declare type Sections<T> = Record<string, Methods<T>>;
declare type Methods<T> = Record<string, T>;
/**
 * @description Takes a decorated api section (e.g. api.tx) and augment it with the details. It does not override what is
 * already available, but rather just adds new missing items into the result object.
 * @internal
 */
export declare function augmentObject<T>(prefix: string | null, src: Sections<T>, dst: Sections<T>, fromEmpty?: boolean): Sections<T>;
export {};
