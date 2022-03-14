/// <reference types="node" />
import type { Server } from 'mock-socket';
import type { Constructor } from '@polkadot/types/types';
export interface Global extends NodeJS.Global {
    WebSocket: Constructor<WebSocket>;
    fetch: any;
}
export interface Mock {
    body: Record<string, any>;
    requests: number;
    server: Server;
    done: () => Record<string, unknown>;
}
export declare type MockStateSubscriptionCallback = (error: Error | null, value: any) => void;
export declare type MockStateSubscriptions = Record<string, {
    callbacks: Record<number, MockStateSubscriptionCallback>;
    lastValue: any;
}>;
export declare type MockStateDb = Record<string, Uint8Array>;
export declare type MockStateRequests = Record<string, (db: MockStateDb, params: any[]) => string>;
