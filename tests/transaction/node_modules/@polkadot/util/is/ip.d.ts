declare type IpTypes = 'v4' | 'v6';
/**
 * @name isIp
 * @summary Tests if the value is a valid IP address
 * @description
 * Checks to see if the value is a valid IP address. Optionally check for either v4/v6
 * @example
 * <BR>
 *
 * ```javascript
 * import { isIp } from '@polkadot/util';
 *
 * isIp('192.168.0.1')); // => true
 * isIp('1:2:3:4:5:6:7:8'); // => true
 * isIp('192.168.0.1', 'v6')); // => false
 * isIp('1:2:3:4:5:6:7:8', 'v4'); // => false
 * ```
 */
export declare function isIp(value: string, type?: IpTypes): boolean;
export {};
