export interface DeriveCache {
    del: (key: string) => void;
    forEach: (cb: (key: string, value: any) => void) => void;
    get: <T = any>(key: string) => T | undefined;
    set: (key: string, value: any) => void;
}
