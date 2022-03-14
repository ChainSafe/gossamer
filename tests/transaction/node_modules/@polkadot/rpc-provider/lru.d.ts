export declare class LRUCache {
    #private;
    readonly capacity: number;
    constructor(capacity?: number);
    get length(): number;
    get lengthData(): number;
    get lengthRefs(): number;
    entries(): [string, unknown][];
    keys(): string[];
    get<T>(key: string): T | null;
    set<T>(key: string, value: T): void;
}
