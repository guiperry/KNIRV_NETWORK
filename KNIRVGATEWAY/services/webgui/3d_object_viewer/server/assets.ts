// assets.ts - Asset management

import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';
import { Object3D, RPCAsset } from './types';
import { Mutex, logger } from './utils';
import { db, dbMutex } from './db';

// Global asset variables
export const objects3D: { [key: string]: Object3D } = {};
export const objectsMutex = new Mutex();
export const objectsDir: string = path.join(process.cwd(), "public/assets/models");

// Scan objects directory for 3D objects
export function scanObjectsDirectory(): void {
    logger.log(`Scanning objects directory: ${objectsDir}...`);
    
    try {
        // Create directory if it doesn't exist
        if (!fs.existsSync(objectsDir)) {
            fs.mkdirSync(objectsDir, { recursive: true });
            logger.log(`Created objects directory: ${objectsDir}`);
            return;
        }
        
        // Read directory contents
        const files = fs.readdirSync(objectsDir);
        logger.log(`Found ${files.length} files in objects directory`);
        
        // Process each file
        for (const file of files) {
            const filePath = path.join(objectsDir, file);
            const stats = fs.statSync(filePath);
            
            if (stats.isFile()) {
                const fileExt = path.extname(file).toLowerCase();
                let objectType = "";
                
                // Determine object type based on extension
                if (fileExt === '.gltf' || fileExt === '.glb') {
                    objectType = fileExt.substring(1); // Remove the dot
                } else if (fileExt === '.usdz') {
                    objectType = 'usdz';
                } else if (fileExt === '.md') {
                    objectType = 'markdown';
                } else {
                    // Skip unsupported file types
                    logger.log(`Skipping unsupported file type: ${file}`);
                    continue;
                }
                
                logger.log(`Processing file: ${file} (${objectType})`);
                
                // Create object entry
                const objectId = crypto.createHash('md5').update(filePath).digest('hex');
                const object: Object3D = {
                    id: objectId,
                    name: path.basename(file, fileExt),
                    file_path: `/assets/models/${file}`, // Use web path instead of filesystem path
                    object_type: objectType,
                    size: stats.size,
                    last_modified: stats.mtimeMs
                };
                
                // Add to objects collection
                objects3D[objectId] = object;
            }
        }
        
        logger.log(`Scanned objects directory: found ${Object.keys(objects3D).length} objects`);
    } catch (error: unknown) {
        if (error instanceof Error) {
            logger.warn(`Failed to scan objects directory: ${error.message}`);
        } else {
            logger.warn(`Failed to scan objects directory`);
        }
    }
}

// getAssetLookupFunction looks up an asset by its ID in the database or storage system
export async function getAssetLookupFunction(assetID: string): Promise<RPCAsset | null> {
    // Validate input
    if (!assetID) {
        throw new Error("asset ID cannot be empty");
    }

    // Check if the asset exists in our database
    await dbMutex.lock();
    try {
        // If we have the asset in our database, return it
        if (db.assets[assetID]) {
            const asset = db.assets[assetID];
            let author = "Unknown";
            let version = 1;
            let license = "Unspecified";
            let contentLocation = "";

            if (asset.data["author"] && typeof asset.data["author"] === 'string') {
                author = asset.data["author"];
            }

            if (asset.data["version"] && typeof asset.data["version"] === 'number') {
                version = asset.data["version"];
            }

            if (asset.data["license"] && typeof asset.data["license"] === 'string') {
                license = asset.data["license"];
            }
            if (asset.data["contentLocation"] && typeof asset.data["contentLocation"] === 'string') {
                contentLocation = asset.data["contentLocation"];
            }

            return {
                AssetID: asset.id,
                Author: author,
                Version: version,
                License: license,
                ContentLocation: contentLocation,
            };
        }
    } finally {
        dbMutex.unlock();
    }


    // For testing purposes, return a mock asset for the sample ID
    if (assetID === "a1b2c3d4e5f6g7h8i9j0") {
        return {
            AssetID: assetID,
            Author: "John Doe",
            Version: 1,
            License: "CC BY 4.0",
            ContentLocation: "ipfs://QmSampleCID",
        };
    }

    // Log the lookup attempt
    logger.log(`Asset lookup failed for ID: ${assetID}`);

    // If we don't have the asset, return null
    return null; // Or throw error if you want to handle not found as an error
}