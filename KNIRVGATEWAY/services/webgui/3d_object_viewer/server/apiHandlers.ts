// apiHandlers.ts - HTTP API request handlers

import * as http from 'http';
import * as util from 'util';
import * as child_process from 'child_process';
import { RegisterContentRequest, AddViewerRequest, RegisterContentResponse, IsViewerAllowedResponse } from './types';
import { logger } from './utils';

export async function registerContentHandler(req: http.IncomingMessage, res: http.ServerResponse): Promise<void> {
    try {
        let body = '';
        req.on('data', (chunk: Buffer) => {
            body += chunk.toString(); // convert Buffer to string
        });

        req.on('end', async () => {
            try {
                const parsedBody: RegisterContentRequest = JSON.parse(body);
                const { encryptedContentLocation, encryptionKeyHash, contentID } = parsedBody;

                // Log the incoming request
                logger.log(`Register Content Request received for Content ID: ${contentID}`);

                // Construct the command to execute the custom blockchain's RPC method
                const command = 'your-blockchain-cli'; // Replace with the actual command
                const args = [
                    'register-content',
                    '--content-id', contentID,
                    '--location', encryptedContentLocation,
                    '--key-hash', encryptionKeyHash,
                ];

                // Execute the command
                const { stdout, stderr } = await util.promisify(child_process.execFile)(command, args);

                // Log the output from the command
                logger.log(`Command Output: ${stdout}`);
                if (stderr) {
                    logger.log(`Command Error: ${stderr}`);
                }

                // Parse the output to construct the RPC response
                const commandOutput = JSON.parse(stdout); // Assuming the command returns JSON
                const rpcResponse: RegisterContentResponse = {
                    success: commandOutput.success,
                    message: commandOutput.message,
                };

                // Send the RPC response back to the client
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify(rpcResponse));

            } catch (parseError) {
                // Handle JSON parsing errors
                logger.log(`Error parsing JSON request body: ${parseError}`);
                res.writeHead(400, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ success: false, message: 'Invalid JSON' }));
            }
        });
    } catch (error: unknown) {
        // Handle unexpected errors
        if (error instanceof Error) {
            logger.log(`Unexpected error in registerContentHandler: ${error.message}`);
        } else {
            logger.log(`Unexpected error in registerContentHandler`);
        }
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ success: false, message: 'Internal Server Error' }));
    }
}

export const handleViewerAuthorization = async (req: http.IncomingMessage, res: http.ServerResponse): Promise<void> => {
    try {
        let body = '';
        req.on('data', (chunk: Buffer) => {
            body += chunk.toString();
        });

        req.on('end', async () => {
            try {
                const parsedBody: AddViewerRequest = JSON.parse(body);
                const { contentID, viewerAddress } = parsedBody;

                // Log the incoming request
                logger.log(`Verify Viewer Request received for Content ID: ${contentID} and Viewer Address: ${viewerAddress}`);

                // Construct the command to execute the custom blockchain's RPC method
                const command = 'your-blockchain-cli'; // Replace with the actual command
                const args = [
                    'verify-viewer',
                    '--content-id', contentID,
                    '--viewer-address', viewerAddress,
                ];

                // Execute the command
                const { stdout, stderr } = await util.promisify(child_process.execFile)(command, args);

                // Log the output from the command
                logger.log(`Command Output: ${stdout}`);
                if (stderr) {
                    logger.log(`Command Error: ${stderr}`);
                }

                // Parse the output to construct the RPC response
                const commandOutput = JSON.parse(stdout); // Assuming the command returns JSON
                const rpcResponse: IsViewerAllowedResponse = {
                    allowed: commandOutput.allowed,
                };

                // Send the RPC response back to the client
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify(rpcResponse));

            } catch (parseError) {
                // Handle JSON parsing errors
                logger.log(`Error parsing JSON request body: ${parseError}`);
                res.writeHead(400, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ allowed: false })); // Return false for invalid JSON
            }
        });
    } catch (error: unknown) {
        // Handle unexpected errors
        if (error instanceof Error) {
            logger.log(`Unexpected error in isViewerAllowedHandler: ${error.message}`);
        } else {
            logger.log(`Unexpected error in isViewerAllowedHandler`);
        }
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ allowed: false })); // Return false for internal server error
    }
};

export const handleViewerAddition = async (req: http.IncomingMessage, res: http.ServerResponse): Promise<void> => {
    try {
        let body = '';
        req.on('data', (chunk: Buffer) => {
            body += chunk.toString();
        });

        req.on('end', async () => {
            try {
                const parsedBody: AddViewerRequest = JSON.parse(body);
                const { contentID, viewerAddress } = parsedBody;

                // Log the incoming request
                logger.log(`Add Viewer Request received for Content ID: ${contentID} and Viewer Address: ${viewerAddress}`);

                // Construct the command to execute the custom blockchain's RPC method
                const command = 'your-blockchain-cli'; // Replace with the actual command
                const args = [
                    'add-viewer',
                    '--content-id', contentID,
                    '--viewer-address', viewerAddress,
                ];

                // Execute the command
                const { stdout, stderr } = await util.promisify(child_process.execFile)(command, args);

                // Log the output from the command
                logger.log(`Command Output: ${stdout}`);
                if (stderr) {
                    logger.log(`Command Error: ${stderr}`);
                }

                // Parse the output to construct the RPC response
                const commandOutput = JSON.parse(stdout); // Assuming the command returns JSON
                const rpcResponse: RegisterContentResponse = {
                    success: commandOutput.success,
                    message: commandOutput.message,
                };

                // Send the RPC response back to the client
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify(rpcResponse));

            } catch (parseError) {
                // Handle JSON parsing errors
                logger.log(`Error parsing JSON request body: ${parseError}`);
                res.writeHead(400, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ success: false, message: 'Invalid JSON' }));
            }
        });
    } catch (error: unknown) {
        // Handle unexpected errors
        if (error instanceof Error) {
            logger.log(`Unexpected error in addViewerHandler: ${error.message}`);
        } else {
            logger.log(`Unexpected error in addViewerHandler`);
        }
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ success: false, message: 'Internal Server Error' }));
    }
};

export async function getContentMetadataHandler(req: http.IncomingMessage, res: http.ServerResponse): Promise<void> {
    try {
        let body = '';
        req.on('data', (chunk: Buffer) => {
            body += chunk.toString();
        });

        req.on('end', async () => {
            try {
                const parsedBody: AddViewerRequest = JSON.parse(body);
                const { contentID } = parsedBody;

                // Log the incoming request
                logger.log(`Get Content Metadata Request received for Content ID: ${contentID}`);

                // Construct the command to execute the custom blockchain's RPC method
                const command = 'your-blockchain-cli'; // Replace with the actual command
                const args = [
                    'get-content-metadata',
                    '--content-id', contentID,
                ];

                // Execute the command
                const { stdout, stderr } = await util.promisify(child_process.execFile)(command, args);

                // Log the output from the command
                logger.log(`Command Output: ${stdout}`);
                if (stderr) {
                    logger.log(`Command Error: ${stderr}`);
                }

                // Parse the output to construct the RPC response
                const commandOutput = JSON.parse(stdout); // Assuming the command returns JSON
                const rpcResponse: RegisterContentResponse = {
                    success: commandOutput.success,
                    message: commandOutput.message,
                };

                // Send the RPC response back to the client
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify(rpcResponse));

            } catch (parseError) {
                // Handle JSON parsing errors
                logger.log(`Error parsing JSON request body: ${parseError}`);
                res.writeHead(400, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ success: false, message: 'Invalid JSON' }));
            }
        });
    } catch (error: unknown) {
        // Handle unexpected errors
        if (error instanceof Error) {
            logger.log(`Unexpected error in addViewerHandler: ${error.message}`);
        } else {
            logger.log(`Unexpected error in addViewerHandler`);
        }
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ success: false, message: 'Internal Server Error' }));
    }
};
