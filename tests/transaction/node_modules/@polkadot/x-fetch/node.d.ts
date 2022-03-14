export { packageInfo } from './packageInfo';
export declare const fetch: {
    (input: RequestInfo, init?: RequestInit | undefined): Promise<Response>;
    (input: RequestInfo, init?: RequestInit | undefined): Promise<Response>;
} & typeof globalThis.fetch;
