// p2p_network.p - P2P Network State Machine for KNIRVROUTER
// Models peer-to-peer networking including discovery, routing, and messaging
// Based on: KNIRVROUTER P2P implementation

machine P2PNetworkMachine {
    // State variables - Peer management
    var peers: map[NodeID, PeerInfo];
    var connectedPeers: set[NodeID];
    var pendingConnections: set[NodeID];
    var bannedPeers: set[NodeID];

    // State variables - Routing
    var routingTable: map[NodeID, RouteEntry];
    var messageQueue: seq[RoutedMessage];
    var deliveredMessages: set[UUID];

    // State variables - Network state
    var localNode: NodeID;
    var networkPartitioned: bool;
    var lastTopologyUpdate: Timestamp;

    // Configuration
    var maxPeers: int;
    var maxMessageTTL: int;
    var connectionTimeout: int;
    var reputationThreshold: int;

    // Counters
    var messageCounter: int;

    // Temporary variables for event handlers
    var tmpI: int;
    var tmpPeer: PeerInfo;
    var tmpRoute: RouteEntry;
    var tmpMessage: RoutedMessage;
    var tmpQueryRoute: seq[NodeID];
    var tmpEntry: RouteEntry;
    var tmpExistingMetric: int;
    var tmpMsgID: UUID;
    var tmpReachedNodes: int;
    var tmpInOurPartition: bool;
    var tmpOurPartition: seq[NodeID];
    var tmpLargestSize: int;
    var tmpLargestIdx: int;
    var tmpPartition: seq[NodeID];
    var tmpJ: int;
    var failure_data: PeerConnectionFailedData;
    var tmpBroadcastData: BroadcastDeliveredData;
    var allRoutingTableKeys: seq[NodeID];
    var tmpNodeID: NodeID;

    start state Init {
        entry {
            peers = default(map[NodeID, PeerInfo]);
            connectedPeers = default(set[NodeID]);
            pendingConnections = default(set[NodeID]);
            bannedPeers = default(set[NodeID]);

            routingTable = default(map[NodeID, RouteEntry]);
            messageQueue = default(seq[RoutedMessage]);
            deliveredMessages = default(set[UUID]);

            localNode.id = "local_node";
            localNode.publicKey = default(seq[int]);
            localNode.nodeType.typeName = "full";
            networkPartitioned = false;
            lastTopologyUpdate.milliseconds = 0;

            maxPeers = 50;
            maxMessageTTL = 10;
            connectionTimeout = 30000;  // 30 seconds
            reputationThreshold = 50;

            messageCounter = 0;
        }

        on eComponentStart do {
            goto Discovering;
        }

        on eNetworkStart do {
            goto Discovering;
        }
    }

    state Discovering {
        entry {
            // Initiate peer discovery
            announce eDiscoverPeers;
        }

        on ePeersDiscovered do (discoveredPeers: seq[PeerInfo]) {
            tmpI = 0;
            while (tmpI < sizeof(discoveredPeers)) {
                tmpPeer = discoveredPeers[tmpI];

                // Add to known peers if not banned
                if (!(tmpPeer.nodeID in bannedPeers)) {
                    peers[tmpPeer.nodeID] = tmpPeer;

                    // Attempt connection if we have capacity
                    if (sizeof(connectedPeers) < maxPeers) {
                        send this, eConnectToPeer, tmpPeer;
                    }
                }

                tmpI = tmpI + 1;
            }

            goto Active;
        }

        // Timeout - proceed anyway
        on eNetworkShutdown do {
            goto Halted;
        }
    }

    state Active {
        // =====================================================================
        // PEER CONNECTION MANAGEMENT
        // =====================================================================

        on eConnectToPeer do (peer: PeerInfo) {
            // Check if already connected or pending
            // if (peer.nodeID in connectedPeers) {
            //     return;
            // }

            // if (peer.nodeID in pendingConnections) {
            //     return;
            // }

            // // Check if banned
            // if (peer.nodeID in bannedPeers) {
            //     failure_data = (peer = peer, reason = "Peer is banned");
            //     send this, ePeerConnectionFailed, failure_data;
            //     return;
            // }

            // // Check capacity
            // if (sizeof(connectedPeers) >= maxPeers) {
            //     failure_data = (peer = peer, reason = "Max peers reached");
            //     send this, ePeerConnectionFailed, failure_data;
            //     return;
            // }

            // // Check reputation
            // if (peer.reputation < reputationThreshold) {
            //     failure_data = (peer = peer, reason = "Low reputation");
            //     send this, ePeerConnectionFailed, failure_data;
            //     return;
            // }

            // // Mark as pending
            // pendingConnections = pendingConnections + {peer.nodeID};

            // // Simulate connection (in real system, would do handshake)
            // // Auto-complete connection
            // remove peer.nodeID from pendingConnections;
            // add peer.nodeID to connectedPeers;
            // peers[peer.nodeID] = peer;

            // // Add to routing table
            // tmpRoute.destination = peer.nodeID;
            // tmpRoute.nextHop = peer.nodeID;  // Direct connection
            // tmpRoute.metric = 1;
            // tmpRoute.lastUpdated.milliseconds = 0;
            // tmpRoute.status.status = "active";
            // routingTable[peer.nodeID] = tmpRoute;

            // announce ePeerConnected, peer;
            // send this, ePeerConnected, peer;
        }

        on eDisconnectPeer do (nodeID: NodeID) {
            if (nodeID in connectedPeers) {
                // remove nodeID from connectedPeers;

                // Update routing table
                if (nodeID in routingTable) {
                    tmpRoute = routingTable[nodeID];
                    tmpRoute.status.status = "unreachable";
                    routingTable[nodeID] = tmpRoute;
                }

                send this, ePeerDisconnected, nodeID;
            }
        }

        // =====================================================================
        // MESSAGE ROUTING
        // =====================================================================

        /*
        on eRouteMessage do (message: RoutedMessage) {
            tmpMessage = message;

            // Check TTL
            if (tmpMessage.ttl <= 0) {
                send this, eMessageRoutingFailed, (tmpMessage.id, "TTL expired");
                return;
            }

            // Check if already delivered
            if (tmpMessage.id in deliveredMessages) {
                return;  // Duplicate
            }

            // Check if we are the destination
            if (tmpMessage.destination.id == localNode.id) {
                                                                deliveredMessages = deliveredMessages + {tmpMessage.id};
                                                                announce eMessageRouted, (tmpMessage.id, sizeof(tmpMessage.hops));
                                send this, eMessageRouted, (tmpMessage.id, sizeof(tmpMessage.hops));
                return;
            }

            // Find route
            if (tmpMessage.destination in routingTable) {
                tmpRoute = routingTable[tmpMessage.destination];

                if (tmpRoute.status.status == "active") {
                    // Forward message
                    tmpMessage.ttl = tmpMessage.ttl - 1;
                    tmpMessage.hops += (sizeof(tmpMessage.hops), localNode);

                    // In real system, would send to next hop
                    announce eMessageRouted, (tmpMessage.id, sizeof(tmpMessage.hops));
                    send this, eMessageRouted, (tmpMessage.id, sizeof(tmpMessage.hops));
                    return;
                }
            }

            // No route found - try flooding
            if (sizeof(connectedPeers) > 0) {
                tmpMessage.ttl = tmpMessage.ttl - 1;
                tmpMessage.hops += (sizeof(tmpMessage.hops), localNode);

                // Broadcast to all peers
                announce eMessageRouted, (tmpMessage.id, sizeof(tmpMessage.hops));
                send this, eMessageRouted, (tmpMessage.id, sizeof(tmpMessage.hops));
            } else {
                send this, eMessageRoutingFailed, (tmpMessage.id, "No route and no peers");
            }
        }
        */

        on eQueryRoute do (destination: NodeID) {
            tmpQueryRoute = default(seq[NodeID]);

            if (destination in routingTable) {
                tmpEntry = routingTable[destination];

                if (tmpEntry.status.status == "active") {
                    tmpQueryRoute += (0, localNode);
                    tmpQueryRoute += (1, tmpEntry.nextHop);
                    tmpQueryRoute += (2, destination);
                }
            }

            send this, eRouteQueryResult, tmpQueryRoute;
        }

        // =====================================================================
        // ROUTING TABLE UPDATES
        // =====================================================================

        on eUpdateRoute do (routeEntry: RouteEntry) {
            // Update routing table with new information
            tmpExistingMetric = 999999;

            if (routeEntry.destination in routingTable) {
                tmpExistingMetric = routingTable[routeEntry.destination].metric;
            }

            // Only update if better route
            if (routeEntry.metric < tmpExistingMetric) {
                routingTable[routeEntry.destination] = routeEntry;
                send this, eRouteUpdated, routeEntry;
            }
        }

        // =====================================================================
        // BROADCAST
        // =====================================================================

        on eBroadcastMessage do (payload: (
            messageType: string,
            msgPayload: seq[int],
            ttl: int
        )) {
            messageCounter = messageCounter + 1;
            tmpMsgID.value = format("msg_{0}", messageCounter);

            tmpReachedNodes = sizeof(connectedPeers);

            // In real system, would actually broadcast
            // For model, just count connected peers
            tmpBroadcastData.messageID = tmpMsgID;
            tmpBroadcastData.deliveryCount = tmpReachedNodes;
            announce eBroadcastDelivered, tmpBroadcastData;
            send this, eBroadcastDelivered, tmpBroadcastData;
        }

        // =====================================================================
        // NETWORK PARTITIONING
        // =====================================================================

        on eNetworkPartitionDetected do (partitions: seq[seq[NodeID]]) {
            networkPartitioned = true;

            // Mark routes to other partitions as unreachable
            allRoutingTableKeys = keys(routingTable);
            tmpI = 0;
            while (tmpI < sizeof(allRoutingTableKeys)) {
                tmpNodeID = allRoutingTableKeys[tmpI];
                tmpInOurPartition = false;

                // Check if node is in our partition (first partition)
                if (sizeof(partitions) > 0) {
                    tmpOurPartition = partitions[0];

                    tmpJ = 0; // Reusing tmpJ for inner loop
                    while (tmpJ < sizeof(tmpOurPartition)) {
                        if (tmpOurPartition[tmpJ].id == tmpNodeID.id) {
                            tmpInOurPartition = true;
                            break;
                        }
                        tmpJ = tmpJ + 1;
                    }
                }

                if (!tmpInOurPartition) {
                    tmpRoute = routingTable[tmpNodeID];
                    tmpRoute.status.status = "unreachable";
                    routingTable[tmpNodeID] = tmpRoute;
                }
                tmpI = tmpI + 1; // Increment tmpI
            }

            announce eMonitorViolation, (monitorName = "NetworkTopology", violationType = "Network partition detected", details = default(map[string, any]));
        }

        on eNetworkHealed do (timestamp: Timestamp) {
            networkPartitioned = false;

            // Re-enable all routes
            allRoutingTableKeys = keys(routingTable); // Reusing the existing machine-level variable
            tmpI = 0;
            while (tmpI < sizeof(allRoutingTableKeys)) {
                tmpNodeID = allRoutingTableKeys[tmpI]; // Reusing the existing machine-level variable tmpNodeID
                if (tmpNodeID in connectedPeers) {
                    tmpRoute = routingTable[tmpNodeID];
                    tmpRoute.status.status = "active";
                    routingTable[tmpNodeID] = tmpRoute;
                }
                tmpI = tmpI + 1;
            }
        }

        // =====================================================================
        // PEER DISCOVERY (CONTINUOUS)
        // =====================================================================

        on eDiscoverPeers do {
            // Trigger peer discovery
            // In real system, would query DHT or bootstrap nodes
            announce ePeersDiscovered, default(seq[PeerInfo]);
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            goto Halted;
        }
    }

    state Halted {
        on eEmergencyResume do {
            goto Active;
        }
    }

    // Helper functions
    fun GetConnectedPeerCount(): int {
        return sizeof(connectedPeers);
    }

    fun GetRouteCount(): int {
        return sizeof(routingTable);
    }

    fun IsNetworkPartitioned(): bool {
        return networkPartitioned;
    }

    fun GetPeerReputation(nodeID: NodeID): int {
        if (nodeID in peers) {
            return peers[nodeID].reputation;
        }
        return 0;
    }
}
