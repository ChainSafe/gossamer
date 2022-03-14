export { packageInfo } from './packageInfo';
export declare const WebSocket: {
    new (url: string | URL, protocols?: string | string[] | undefined): WebSocket;
    prototype: WebSocket;
    readonly CLOSED: number;
    readonly CLOSING: number;
    readonly CONNECTING: number;
    readonly OPEN: number;
};
