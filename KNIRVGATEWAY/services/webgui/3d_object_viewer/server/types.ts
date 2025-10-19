// types.ts - Contains all interface definitions for the application

// Asset type definition
export interface AssetData {
    author?: string;
    version?: number;
    license?: string;
    contentLocation?: string;
    [key: string]: unknown; // For any additional dynamic properties
}

export interface Asset {
    id: string;
    name: string;
    data: AssetData;
    created_at: Date;
    transaction: string;
    asset_type?: string; // e.g., "3d", "image", "document"
    file_url?: string;   // URL or path to the asset file
    object_type?: string; // Type of object: "gltf", "glb", "usdz", "markdown"
    uploaded_at?: Date;
}

export interface RPCParams {
    [key: string]: unknown;
}

export interface Args {
    a: number;
    b: number;
}

export interface GetAssetDetailsArgs {
    assetID: string;
}

// Block represents a block in the blockchain with BigchainDB-inspired structure
export interface Block {
    index: number;
    id: string;
    data: string;
    timestamp: Date;
    prev_hash: string;
    hash: string;
    signature?: string; // Optional signature
}

// BigchainDB represents our database structure
export interface BigchainDB {
    blocks: Block[];
    assets: { [key: string]: Asset };
    transactions: { [key: string]: Transaction };
}

// Object3D represents a 3D object asset
export interface Object3D {
    id: string;
    name: string;
    description?: string;
    file_path?: string; // Path to the object file (server-side)
    url?: string;      // Object URL (browser-side)
    thumbnail_path?: string;
    uploaded_at?: Date;
    asset_id?: string; // Reference to the blockchain asset
    object_type?: string; // Type of object: "gltf", "glb", "usdz", "markdown"
    size?: number;     // File size in bytes
    last_modified?: number; // Timestamp of last modification
}

// Transaction represents a transaction in the system
export interface Transaction {
    id: string;
    operation: string; // CREATE, TRANSFER
    asset: string;     // Asset ID
    metadata: string;
    timestamp: Date;
}

// Configuration (Read from environment variables or .env file)
export interface Config {
    RPCEndpoint: string;
    ServiceAddress: string; //The address that the backend will use
    Port: string;
}

// RPC Configuration
export interface RPCConfig {
    Port: string;
}

// registerContentRequest defines the structure for the /register-content endpoint.
export interface RegisterContentRequest {
    encryptedContentLocation: string;
    encryptionKeyHash: string;
    contentID: string;
}

// addViewerRequest defines the structure for the /add-viewer endpoint
export interface AddViewerRequest {
    contentID: string;
    viewerAddress: string;
}

// registerContentResponse mirrors the structure of your custom blockchain's RPC response.
export interface RegisterContentResponse {
    success: boolean;
    message: string;
}

// isViewerAllowedResponse mirrors the structure of your custom blockchain's RPC response
export interface IsViewerAllowedResponse {
    allowed: boolean;
}

// Quotient represents the result of the division RPC method.
export interface Quotient {
    quo: number;
    rem: number;
}

// AssetDetails represents the details of a 3D asset.
export interface AssetDetails {
    AssetID: string;
    Author: string;
    Version: number;
    License: string;
    ContentLocation: string;
}

// ClientAssetDetails represents the client-side version of AssetDetails
export interface ClientAssetDetails {
    AssetID: string;
    Author: string;
    Version: number;
    License: string;
    ContentLocation: string;
}

// RPCAsset represents an asset in the RPC system
export interface RPCAsset {
    AssetID: string;
    Author: string;
    Version: number;
    License: string;
    ContentLocation: string;
    // ... other fields ...
}

// JSON-RPC types
export type JSONRPCID = string | number | null;

export interface JSONRPCRequest {
    jsonrpc: '2.0';
    method: string;
    params?: unknown[];
    id?: JSONRPCID;
}

export interface JSONRPCResponse {
    jsonrpc: '2.0';
    result?: unknown;
    error?: {
        code: number;
        message: string;
        data?: unknown;
    };
    id: JSONRPCID;
}