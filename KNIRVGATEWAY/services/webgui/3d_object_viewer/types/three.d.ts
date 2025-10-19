import 'three';

declare module 'three' {
    interface Object3D {
        object_type: string;
        id: string;
        uuid: string;
        name: string;
    }
}

type JSONRPCID = string | number | null;

interface GUILogger {
    error(message: string): void;
    log(message: string): void;
    warn(message: string): void;
    info(message: string): void;
}

interface JSONRPCRequest {
    jsonrpc: '2.0';
    method: string;
    params?: unknown[];
    id?: JSONRPCID;
}

interface JSONRPCResponse {
    jsonrpc: '2.0';
    result?: unknown;
    error?: {
        code: number;
        message: string;
        data?: unknown;
    };
    id: JSONRPCID;
}