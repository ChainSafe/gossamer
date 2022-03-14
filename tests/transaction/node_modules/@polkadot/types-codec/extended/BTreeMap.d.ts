import type { Codec, CodecClass } from '../types';
import { CodecMap } from './Map';
export declare class BTreeMap<K extends Codec = Codec, V extends Codec = Codec> extends CodecMap<K, V> {
    static with<K extends Codec, V extends Codec>(keyType: CodecClass<K> | string, valType: CodecClass<V> | string): CodecClass<CodecMap<K, V>>;
}
