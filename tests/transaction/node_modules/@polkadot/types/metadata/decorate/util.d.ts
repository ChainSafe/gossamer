import type { Text } from '@polkadot/types-codec';
declare type Name = string | Text;
interface Named {
    name: Name;
}
export declare const objectNameToCamel: (n: Named) => string;
export declare const objectNameToString: (n: Named) => string;
export {};
