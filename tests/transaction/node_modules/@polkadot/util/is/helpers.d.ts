export declare function isOn<T>(...fns: (keyof T)[]): (value?: unknown) => value is T;
export declare function isOnObject<T>(...fns: (keyof T)[]): (value?: unknown) => value is T;
