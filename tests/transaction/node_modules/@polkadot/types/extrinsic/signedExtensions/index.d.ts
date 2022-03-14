import type { ExtDef, ExtInfo, ExtTypes } from './types';
export declare const allExtensions: ExtDef;
export declare const fallbackExtensions: string[];
export declare function findUnknownExtensions(extensions: string[], userExtensions?: ExtDef): string[];
export declare function expandExtensionTypes(extensions: string[], type: keyof ExtInfo, userExtensions?: ExtDef): ExtTypes;
