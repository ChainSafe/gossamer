export declare enum TypeDefInfo {
    BTreeMap = 0,
    BTreeSet = 1,
    Compact = 2,
    DoNotConstruct = 3,
    Enum = 4,
    HashMap = 5,
    Int = 6,
    Linkage = 7,
    Null = 8,
    Option = 9,
    Plain = 10,
    Range = 11,
    RangeInclusive = 12,
    Result = 13,
    Set = 14,
    Si = 15,
    Struct = 16,
    Tuple = 17,
    UInt = 18,
    Vec = 19,
    VecFixed = 20,
    WrapperKeepOpaque = 21,
    WrapperOpaque = 22
}
export interface TypeDef {
    alias?: Map<string, string>;
    displayName?: string;
    docs?: string[];
    fallbackType?: string;
    info: TypeDefInfo;
    index?: number;
    isFromSi?: boolean;
    length?: number;
    lookupIndex?: number;
    lookupName?: string;
    lookupNameRoot?: string;
    name?: string;
    namespace?: string;
    sub?: TypeDef | TypeDef[];
    type: string;
    typeName?: string;
}
