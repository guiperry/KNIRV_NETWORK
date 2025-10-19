// rpcService.ts - RPC service implementation

import * as http from 'http';
import * as url from 'url';
import {
    Args,
    GetAssetDetailsArgs,
    Quotient,
    ClientAssetDetails,
    AssetDetails,
    RPCParams
} from './types';
import { logger } from './utils';
import { getAssetLookupFunction } from './assets';
import { rpc_config } from './config';

// Arith is the receiver type for our RPC methods.
export class Arith {
    // Multiply is an RPC method that multiplies two integers.
    async multiply(args: Args): Promise<number> {
        const reply = args.a * args.b;
        logger.log(`Multiply: ${args.a} * ${args.b} = ${reply}`);
        return reply;
    }

    // Divide is an RPC method that divides two integers.
    async divide(args: Args): Promise<Quotient> {
        if (args.b === 0) {
            throw new Error("divide by zero");
        }
        const quo = Math.floor(args.a / args.b); // Go's integer division behavior
        const rem = args.a % args.b;
        logger.log(`Divide: ${args.a} / ${args.b} = ${quo} (rem ${rem})`);
        return { quo, rem };
    }

    // GetAssetDetailsArgs defines the arguments for the GetAssetDetails RPC method.
    async getAssetDetails(args: GetAssetDetailsArgs): Promise<ClientAssetDetails> {
        const assetID = args.assetID;

        // 1. Look up the asset in your database or storage system.
        const asset = await getAssetLookupFunction(assetID);
        if (!asset) {
            throw new Error(`Asset not found: ${assetID}`);
        }

        // 2. Populate the AssetDetails struct with the retrieved information.
        const reply: AssetDetails = {
            AssetID: asset.AssetID,
            Author: asset.Author,
            Version: asset.Version,
            License: asset.License,
            ContentLocation: asset.ContentLocation,
        };

        logger.log(`GetAssetDetails: AssetID=${assetID}, Author=${asset.Author}`);
        return {
            AssetID: reply.AssetID,
            Author: reply.Author,
            Version: reply.Version,
            License: reply.License,
            ContentLocation: reply.ContentLocation,
        };
    }
}

export function startRPCService(): http.Server {
    const rpcPort = parseInt(rpc_config.Port, 10);
    logger.log(`Starting RPC service on port ${rpcPort}...`);

    const rpcRouter = http.createServer(async (req: http.IncomingMessage, res: http.ServerResponse) => {
        const parsedUrl = url.parse(req.url || '', true);
        const pathname = parsedUrl.pathname;

        // Enable CORS
        res.setHeader('Access-Control-Allow-Origin', '*');
        res.setHeader('Access-Control-Request-Method', '*');
        res.setHeader('Access-Control-Allow-Methods', 'OPTIONS, POST');
        res.setHeader('Access-Control-Allow-Headers', '*');

        if (req.method === 'OPTIONS') {
            res.writeHead(200);
            res.end();
            return;
        }

        // Log the incoming request
        logger.log(`Incoming RPC request: ${req.method} ${req.url}`);

        // Only handle POST requests to /rpc
        if (req.method !== 'POST' || pathname !== '/rpc') {
            res.writeHead(404);
            res.end('Not Found');
            return;
        }

        try {
            let body = '';
            req.on('data', (chunk: Buffer) => {
                body += chunk.toString();
            });

            await new Promise<void>((resolve, reject) => {
                req.on('end', async () => {
                    try {
                        const request = JSON.parse(body) as {
                            jsonrpc: string;
                            method: string;
                            params: RPCParams;
                            id: string | number | null;
                        };

                        // Validate the request
                        if (request.jsonrpc !== '2.0' || !request.method) {
                            throw new Error('Invalid JSON-RPC 2.0 request');
                        }

                        // Handle different RPC methods
                        const arith = new Arith();
                        let result;
                        switch (request.method) {
                            case 'Multiply':
                                result = await arith.multiply(request.params as unknown as Args);
                                break;
                            case 'Divide':
                                result = await arith.divide(request.params as unknown as Args);
                                break;
                            case 'GetAssetDetails':
                                result = await arith.getAssetDetails(request.params as unknown as GetAssetDetailsArgs);
                                break;
                            default:
                                throw new Error(`Method not supported: ${request.method}`);
                        }

                        // Send successful response
                        res.writeHead(200, { 'Content-Type': 'application/json' });
                        res.end(JSON.stringify({
                            jsonrpc: '2.0',
                            result: result,
                            id: request.id
                        }));
                        resolve();
                    } catch (error: unknown) {
                        const err = error instanceof Error ? error : new Error(String(error));
                        logger.error(`RPC error: ${err.message}`);
                        res.writeHead(400, { 'Content-Type': 'application/json' });
                        res.end(JSON.stringify({
                            jsonrpc: '2.0',
                            error: {
                                code: -32603,
                                message: err.message
                            },
                            id: null
                        }));
                        resolve();
                    }
                });

                req.on('error', (err: Error) => {
                    logger.error(`Request error: ${err.message}`);
                    res.writeHead(500);
                    res.end('Internal Server Error');
                    reject(err);
                });
            });
        } catch (error: unknown) {
            const err = error instanceof Error ? error : new Error(String(error));
            logger.error(`RPC handler error: ${err.message}`);
            res.writeHead(500, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({
                jsonrpc: '2.0',
                error: {
                    code: -32603,
                    message: 'Internal server error'
                },
                id: null
            }));
        }
    });

    rpcRouter.listen(rpcPort, () => {
        logger.log(`RPC Service listening on port ${rpcPort}`);
    });

    return rpcRouter;
}