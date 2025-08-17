// agent-tunnel-registry/api/uriRoutes.js
const express = require('express');
const registryManager = require('../registry/registryManager');
const config = require('../config');
const { v4: uuidv4 } = require('uuid');
const axios = require('axios');

const router = express.Router();

// Generates a agent:// URI that implies tunneling through this server if the node is tunneled
router.post('/generate', (req, res) => {
    const { devId, resourceType = 'dev', subPath = '' } = req.body; // Node requesting URI generation
    
    if (!devId) {
        return res.status(400).json({ error: 'devId is required for URI generation' });
    }

    const nodeInfo = registryManager.getNodeByPeerId(devId);
    if (!nodeInfo) {
        return res.status(404).json({ error: `Node with devId ${devId} not registered. Register first or connect via control channel.` });
    }

    // Generate a unique ID for this resource
    const resourceSpecificId = uuidv4();

    // Check if this generated ID already exists globally (on blockchain/DHT)
    const idExistsUrl = `http://localhost:${config.goInternalApiPort}/internal/db/idExists?id=${resourceSpecificId}`;
    axios.get(idExistsUrl).then(response => {
        if (response.data && response.data.exists) {
            // If the randomly generated ID exists, try again
            console.warn(`[URI Generate] Generated ID ${resourceSpecificId} already exists, retrying...`);
            router.handle('POST', req, res); // Re-route the request internally
            return;
        }

        // Map the unique resource ID to the node's devId in the local registry
        registryManager.mapTunneledResource(resourceSpecificId, devId);
        
        // Generate and return the URI
        let uri;
        if (nodeInfo.isTunneled) {
            uri = `agent://${config.serverPublicHost}/${resourceSpecificId}.${resourceType}${subPath ? '/' + subPath.replace(/^\//, '') : ''}`;
        } else {
            uri = `agent://${nodeInfo.publicIp || nodeInfo.devId}/${resourceSpecificId}.${resourceType}${subPath ? '/' + subPath.replace(/^\//, '') : ''}`;
        }
        
        // Announce the resource on the DHT
        axios.post(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`, {
            id: resourceSpecificId,
            type: resourceType,
            multiaddress: nodeInfo.isTunneled ? 
                `/ip4/${config.serverPublicHost}/tcp/${config.publicRelayPort}/p2p/${config.relayServerPeerId}/p2p-circuit/p2p/${devId}` :
                `/ip4/${nodeInfo.publicIp}/tcp/${nodeInfo.publicP2pPort}/p2p/${devId}`
        }).catch(err => console.error(`[URI Generate] Failed to announce resource ${resourceSpecificId} on DHT: ${err.message}`));
        
        res.json({
            uri,
            resourceId: resourceSpecificId,
            resourceType,
            subPath,
            directInfo: !nodeInfo.isTunneled ? { ip: nodeInfo.publicIp, port: nodeInfo.publicP2pPort } : null
        });
    }).catch(err => {
        console.warn(`[URI Generate] Go internal API not available (${err.message}), proceeding without ID verification`);

        // Proceed without ID verification for testing/development
        // Map the unique resource ID to the node's devId in the local registry
        registryManager.mapTunneledResource(resourceSpecificId, devId);

        // Generate and return the URI
        let uri;
        if (nodeInfo.isTunneled) {
            uri = `agent://${config.serverPublicHost}/${resourceSpecificId}.${resourceType}${subPath ? '/' + subPath.replace(/^\//, '') : ''}`;
        } else {
            uri = `agent://${nodeInfo.publicIp || nodeInfo.devId}/${resourceSpecificId}.${resourceType}${subPath ? '/' + subPath.replace(/^\//, '') : ''}`;
        }

        // Try to announce the resource on the DHT (optional for testing)
        axios.post(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`, {
            id: resourceSpecificId,
            type: resourceType,
            multiaddress: nodeInfo.isTunneled ?
                `/ip4/${config.serverPublicHost}/tcp/${config.publicRelayPort}/p2p/${config.relayServerPeerId}/p2p-circuit/p2p/${devId}` :
                `/ip4/${nodeInfo.publicIp}/tcp/${nodeInfo.publicP2pPort}/p2p/${devId}`
        }).catch(err => console.warn(`[URI Generate] Failed to announce resource ${resourceSpecificId} on DHT: ${err.message}`));

        res.json({
            uri,
            resourceId: resourceSpecificId,
            resourceType,
            subPath,
            directInfo: !nodeInfo.isTunneled ? { ip: nodeInfo.publicIp, port: nodeInfo.publicP2pPort } : null
        });
    });
});

// Client calls this to understand how to connect given a agent:// URI
router.get('/resolve', (req, res) => {
    const { uri } = req.query;
    if (!uri) {
        return res.status(400).json({ error: 'uri query parameter is required' });
    }

    try {
        // Parse the URI to extract components
        // agent://<bootnode_authority>/<identifier>.<type>/[<subpath>][?<query>]
        const schemePrefix = "agent://";
        if (!uri.startsWith(schemePrefix)) {
            return res.status(400).json({ error: 'Invalid URI scheme' });
        }
        
        const pathPart = uri.substring(schemePrefix.length);
        const firstSlashIndex = pathPart.indexOf('/');
        
        if (firstSlashIndex === -1) {
            return res.status(400).json({ error: 'Invalid URI format: missing path separator' });
        }
        
        const authority = pathPart.substring(0, firstSlashIndex);
        const remainingPath = pathPart.substring(firstSlashIndex + 1);
        
        // Extract identifier and resourceType
        const secondSlashIndex = remainingPath.indexOf('/');
        const identifierAndType = secondSlashIndex !== -1 ? 
            remainingPath.substring(0, secondSlashIndex) : 
            remainingPath;
        
        const lastDotIndex = identifierAndType.lastIndexOf('.');
        if (lastDotIndex === -1) {
            return res.status(400).json({ error: 'URI missing resource type' });
        }
        
        const identifier = identifierAndType.substring(0, lastDotIndex);
        const resourceType = identifierAndType.substring(lastDotIndex + 1);
        
        // Extract subPath and query
        const subPathWithQuery = secondSlashIndex !== -1 ? 
            remainingPath.substring(secondSlashIndex) : 
            '';
        
        // Determine if this is a tunneled resource or direct dev
        let connectionDetails = null;
        
        // Check if this is a resource ID that maps to a node (tunneled or direct)
        const targetPeerIdForResource = registryManager.getPeerIdForTunneledResource(identifier);

        if (targetPeerIdForResource) {
            const nodeInfo = registryManager.getNodeByPeerId(targetPeerIdForResource);
            if (nodeInfo) {
                if (nodeInfo.isTunneled) {
                    connectionDetails = {
                        connectionType: 'TUNNELED',
                        targetPeerId: targetPeerIdForResource,
                        tunnelServerHost: config.serverPublicHost,
                        tunnelServerPort: config.publicRelayPort,
                        relayProtocolInfo: "Send targetPeerId as first line (JSON: {\"targetPeerId\":\"Qm...\"}) after connecting to tunnel server."
                    };
                } else if (nodeInfo.publicIp && nodeInfo.publicP2pPort) {
                    // For direct nodes mapped via resource ID
                    const directMultiaddress = `/ip4/${nodeInfo.publicIp}/tcp/${nodeInfo.publicP2pPort}/p2p/${nodeInfo.devId}`;
                    connectionDetails = {
                        connectionType: 'DIRECT_P2P',
                        targetPeerId: nodeInfo.devId,
                        multiaddress: directMultiaddress,
                    };
                }
            }
        }
        
        // If not tunneled, check if it's a direct dev ID or chain ID
        let nodeInfo = registryManager.getNodeByPeerId(identifier) || 
                      registryManager.getNodeByChainId(identifier);
        
        if (nodeInfo) {
            // Construct connection details based on node info
            if (nodeInfo.isTunneled) {
                // For tunneled nodes, provide relay information
                const relayTransportSegment = `/ip4/${config.serverPublicHost}/tcp/${config.publicRelayPort}/p2p/${config.relayServerPeerId}`;
                const fullRelayMultiaddress = `${relayTransportSegment}/p2p-circuit/p2p/${nodeInfo.devId}`;
                connectionDetails = {
                    connectionType: 'RELAYED_P2P_CIRCUIT',
                    targetPeerId: nodeInfo.devId,
                    multiaddress: fullRelayMultiaddress,
                };
            } else if (nodeInfo.publicIp && nodeInfo.publicP2pPort && nodeInfo.devId) {
                // For directly reachable nodes
                const directMultiaddress = `/ip4/${nodeInfo.publicIp}/tcp/${nodeInfo.publicP2pPort}/p2p/${nodeInfo.devId}`;
                connectionDetails = {
                    connectionType: 'DIRECT_P2P',
                    targetPeerId: nodeInfo.devId,
                    multiaddress: directMultiaddress,
                };
            }
        }
        
        if (connectionDetails === null) {
            // If not a tunneled resource ID handled by this specific bootnode,
            // check the global DHT via the Go process.
            const dhtLookupUrl = `http://localhost:${config.goInternalApiPort}/internal/dht/findResource?id=${identifier}&type=${resourceType}`;
            axios.get(dhtLookupUrl).then(response => {
                if (response.data && response.data.length > 0) {
                    // Found providers on the DHT
                    const provider = response.data[0]; // Take the first provider for now
                    
                    // Extract peer ID and multiaddress from the provider
                    const peerId = provider.ID;
                    const multiaddress = provider.Addrs && provider.Addrs.length > 0 ? provider.Addrs[0] : null;
                    
                    if (peerId && multiaddress) {
                        // Determine connection type based on multiaddress
                        const connectionType = multiaddress.includes('/p2p-circuit/') ? 'RELAYED_P2P_CIRCUIT' : 'DIRECT_P2P';
                        
                        res.json({
                            originalUri: uri,
                            resolvedIdentifier: identifier,
                            resourceType: resourceType,
                            subPathWithQuery: subPathWithQuery,
                            authority: authority,
                            connectionType: connectionType,
                            targetPeerId: peerId,
                            multiaddress: multiaddress
                        });
                    } else {
                        res.status(404).json({ error: `Resource with identifier '${identifier}' found on DHT but has incomplete connection information.` });
                    }
                } else {
                    // If not found on DHT, check the blockchain for capability records
                    const dbLookupUrl = `http://localhost:${config.goInternalApiPort}/internal/db/getCapability?id=${identifier}`;
                    axios.get(dbLookupUrl).then(dbResponse => {
                        if (dbResponse.data) {
                            // Found a capability record on the blockchain
                            const capability = dbResponse.data;
                            
                            // Extract provider information from the capability
                            const providerPeerId = capability.ProviderID || capability.OwnerID;
                            
                            if (providerPeerId) {
                                // Try to find the provider in the local registry
                                const providerInfo = registryManager.getNodeByPeerId(providerPeerId);
                                
                                if (providerInfo) {
                                    // Provider is registered with this bootnode
                                    if (providerInfo.isTunneled) {
                                        // For tunneled providers
                                        const relayTransportSegment = `/ip4/${config.serverPublicHost}/tcp/${config.publicRelayPort}/p2p/${config.relayServerPeerId}`;
                                        const fullRelayMultiaddress = `${relayTransportSegment}/p2p-circuit/p2p/${providerInfo.devId}`;
                                        
                                        res.json({
                                            originalUri: uri,
                                            resolvedIdentifier: identifier,
                                            resourceType: resourceType,
                                            subPathWithQuery: subPathWithQuery,
                                            authority: authority,
                                            connectionType: 'RELAYED_P2P_CIRCUIT',
                                            targetPeerId: providerInfo.devId,
                                            multiaddress: fullRelayMultiaddress
                                        });
                                    } else if (providerInfo.publicIp && providerInfo.publicP2pPort) {
                                        // For directly reachable providers
                                        const directMultiaddress = `/ip4/${providerInfo.publicIp}/tcp/${providerInfo.publicP2pPort}/p2p/${providerInfo.devId}`;
                                        
                                        res.json({
                                            originalUri: uri,
                                            resolvedIdentifier: identifier,
                                            resourceType: resourceType,
                                            subPathWithQuery: subPathWithQuery,
                                            authority: authority,
                                            connectionType: 'DIRECT_P2P',
                                            targetPeerId: providerInfo.devId,
                                            multiaddress: directMultiaddress
                                        });
                                    } else {
                                        res.status(404).json({ error: `Provider for capability '${identifier}' found but has incomplete connection information.` });
                                    }
                                } else {
                                    // Provider not registered with this bootnode, try to find on DHT
                                    const providerLookupUrl = `http://localhost:${config.goInternalApiPort}/internal/dht/findResource?id=${providerPeerId}&type=dev`;
                                    
                                    axios.get(providerLookupUrl).then(providerResponse => {
                                        if (providerResponse.data && providerResponse.data.length > 0) {
                                            // Found provider on DHT
                                            const provider = providerResponse.data[0];
                                            const peerId = provider.ID;
                                            const multiaddress = provider.Addrs && provider.Addrs.length > 0 ? provider.Addrs[0] : null;
                                            
                                            if (peerId && multiaddress) {
                                                const connectionType = multiaddress.includes('/p2p-circuit/') ? 'RELAYED_P2P_CIRCUIT' : 'DIRECT_P2P';
                                                
                                                res.json({
                                                    originalUri: uri,
                                                    resolvedIdentifier: identifier,
                                                    resourceType: resourceType,
                                                    subPathWithQuery: subPathWithQuery,
                                                    authority: authority,
                                                    connectionType: connectionType,
                                                    targetPeerId: peerId,
                                                    multiaddress: multiaddress
                                                });
                                            } else {
                                                res.status(404).json({ error: `Provider for capability '${identifier}' found on DHT but has incomplete connection information.` });
                                            }
                                        } else {
                                            res.status(404).json({ error: `Provider for capability '${identifier}' not found on DHT.` });
                                        }
                                    }).catch(providerError => {
                                        res.status(500).json({ error: `Error looking up provider for capability '${identifier}' on DHT: ${providerError.message}` });
                                    });
                                }
                            } else {
                                res.status(404).json({ error: `Capability '${identifier}' found but has no provider information.` });
                            }
                        } else {
                            res.status(404).json({ error: `Resource with identifier '${identifier}' not found in local registry, DHT, or blockchain.` });
                        }
                    }).catch(dbError => {
                        console.warn(`[URI Resolve] Database lookup failed for '${identifier}': ${dbError.message}`);
                        res.status(404).json({ error: `Resource with identifier '${identifier}' not found in local registry, DHT, or blockchain. Database service unavailable.` });
                    });
                }
            }).catch(dhtError => {
                console.warn(`[URI Resolve] DHT lookup failed for '${identifier}': ${dhtError.message}`);
                res.status(404).json({ error: `Resource with identifier '${identifier}' not found in local registry. DHT service unavailable.` });
            });
            return; // Exit here as the response will be sent asynchronously
        }
        
        // If connectionDetails was set, send the response immediately
        res.json({
            originalUri: uri,
            resolvedIdentifier: identifier,
            resourceType: resourceType,
            subPathWithQuery: subPathWithQuery,
            authority: authority,
            ...connectionDetails
        });
        
    } catch (error) {
        console.error(`[URI Resolve] Error processing URI '${uri}':`, error);
        res.status(500).json({ error: 'Internal server error during URI resolution.' });
    }
});

module.exports = router;