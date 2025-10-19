// server/server.ts
import * as http from 'http';
import * as url from 'url';
import { loadDatabase } from './db';
import { scanObjectsDirectory } from './assets';
import { loadAppConfig, loadRPC_Config } from './config';
import { logger } from './utils';
import { startRPCService } from './rpcService'; // Assuming RPC starts separately or is integrated
import * as apiHandlers from './apiHandlers'; // Import all handlers
import { handleObjectsRequest, handleObjectRequest } from './routes/objects';
import { db } from './db';
import { backendPort } from './config'; // Import the port

// Basic error handler middleware (example)
function handleError(res: http.ServerResponse, statusCode: number, message: string) {
    logger.error(message);
    res.writeHead(statusCode, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ success: false, message }));
}

function initializeCoreSystems() {
    logger.log("Initializing core systems...");
    loadDatabase();
    scanObjectsDirectory();
    logger.log("Core systems initialized.");
}

function startContentHandlerService() {
    const serverPort = backendPort; // Use the loaded port

    const router = http.createServer(async (req: http.IncomingMessage, res: http.ServerResponse) => {
        const parsedUrl = url.parse(req.url || '', true);
        const pathname = parsedUrl.pathname || '';

        // Basic CORS
        res.setHeader('Access-Control-Allow-Origin', '*'); // Be more specific in production
        res.setHeader('Access-Control-Allow-Methods', 'OPTIONS, GET, POST');
        res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

        if (req.method === 'OPTIONS') {
            res.writeHead(204);
            res.end();
            return;
        }

        logger.log(`Content Handler: ${req.method} ${req.url}`);

        try {
            // Handle API routes
            if (pathname.startsWith('/api/')) {
                const apiPath = pathname.substring(5); // Remove '/api/' prefix
                
                if (apiPath === 'objects') {
                    handleObjectsRequest(req, res);
                    return;
                }
                
                if (apiPath.startsWith('objects/')) {
                    const objectId = apiPath.substring(8); // Remove 'objects/' prefix
                    handleObjectRequest(req, res, objectId);
                    return;
                }
                
                if (apiPath === 'blocks') {
                    // Return blocks from the database
                    res.writeHead(200, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify(db.blocks));
                    return;
                }
                
                if (apiPath === 'assets') {
                    // Return assets from the database
                    res.writeHead(200, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify(Object.values(db.assets)));
                    return;
                }
                
                if (apiPath === 'transactions') {
                    // Return transactions from the database
                    res.writeHead(200, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify(Object.values(db.transactions)));
                    return;
                }
                
                // API route not found
                handleError(res, 404, 'API endpoint not found');
                return;
            }
            
            // Handle content handler routes
            switch (pathname) {
                case '/register-content':
                    if (req.method === 'POST') await apiHandlers.registerContentHandler(req, res);
                    else handleError(res, 405, 'Method Not Allowed');
                    break;
                case '/is-viewer-allowed':
                     if (req.method === 'POST') await apiHandlers.handleViewerAuthorization(req, res);
                     else handleError(res, 405, 'Method Not Allowed');
                    break;
                case '/add-viewer':
                     if (req.method === 'POST') await apiHandlers.handleViewerAddition(req, res);
                     else handleError(res, 405, 'Method Not Allowed');
                    break;
                case '/get-content-metadata':
                     if (req.method === 'POST') await apiHandlers.getContentMetadataHandler(req, res);
                     else handleError(res, 405, 'Method Not Allowed');
                    break;
                default:
                    handleError(res, 404, 'Not Found');
            }
        } catch (error: unknown) {
             handleError(res, 500, `Internal Server Error: ${error instanceof Error ? error.message : String(error)}`);
        }
    });

    router.listen(serverPort, () => {
        logger.log(`Content Handler Service listening on port ${serverPort}`);
    });

    return router; // Return the server instance if needed
}

export async function startServer(): Promise<void> {
    loadAppConfig();
    loadRPC_Config(); // Ensure config is loaded first
    initializeCoreSystems();
    startContentHandlerService();
    startRPCService(); // Start RPC service (ensure ports don't clash)
    logger.log("Backend server setup complete.");
    // Keep the process alive (Node.js HTTP servers do this by default)
}
