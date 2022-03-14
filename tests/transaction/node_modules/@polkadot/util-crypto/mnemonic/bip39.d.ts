export declare function mnemonicToSeedSync(mnemonic: string, password?: string): Uint8Array;
export declare function mnemonicToEntropy(mnemonic: string): Uint8Array;
export declare function entropyToMnemonic(entropy: Uint8Array): string;
export declare function generateMnemonic(strength?: number): string;
export declare function validateMnemonic(mnemonic: string): boolean;
