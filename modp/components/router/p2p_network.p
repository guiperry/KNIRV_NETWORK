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

        on ePeersDiscovered do (payload: (peers: seq[PeerInfo])) {
            tmpI = 0;
            while (tmpI < sizeof(payload.peers)) {
                tmpPeer = payload.peers[tmpI];

                // Add to known peers if not banned
                if (!(tmpPeer.nodeID in bannedPeers)) {
                    peers[tmpPeer.nodeID] = tmpPeer;

                    // Attempt connection if we have capacity
                    if (sizeof(connectedPeers) < maxPeers) {
                        send this, eConnectToPeer, (tmpPeer,);
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

        on eConnectToPeer do (payload: (peer: PeerInfo)) {
            // Check if already connected or pending
            if (payload.peer.nodeID in connectedPeers) {
                return;
            }

            if (payload.peer.nodeID in pendingConnections) {
                return;
            }

            // Check if banned
            if (payload.peer.nodeID in bannedPeers) {
                send this, ePeerConnectionFailed, (payload.peer, "Peer is banned");
                return;
            }

            // Check capacity
            if (sizeof(connectedPeers) >= maxPeers) {
                send this, ePeerConnectionFailed, (payload.peer, "Max peers reached");
                return;
            }

            // Check reputation
            if (payload.peer.reputation < reputationThreshold) {
                send this, ePeerConnectionFailed, (payload.peer, "Low reputation");
                return;
            }

            // Mark as pending
            pendingConnections = pendingConnections + (payload.peer.nodeID);

            // Simulate connection (in real system, would do handshake)
            // Auto-complete connection
            pendingConnections = pendingConnections - (payload.peer.nodeID);
            connectedPeers = connectedPeers + (payload.peer.nodeID);
            peers[payload.peer.nodeID] = payload.peer;

            // Add to routing table
            tmpRoute.destination = payload.peer.nodeID;
            tmpRoute.nextHop = payload.peer.nodeID;  // Direct connection
            tmpRoute.metric = 1;
            tmpRoute.lastUpdated.milliseconds = 0;
            tmpRoute.status.status = "active";
            routingTable[payload.peer.nodeID] = tmpRoute;

            announce ePeerConnected, (payload.peer,);
            send this, ePeerConnected, (payload.peer,);
        }

        on eDisconnectPeer do (payload: (nodeID: NodeID)) {
            if (payload.nodeID in connectedPeers) {
                connectedPeers = connectedPeers - (payload.nodeID);

                // Update routing table
                if (payload.nodeID in routingTable) {
                    tmpRoute = routingTable[payload.nodeID];
                    tmpRoute.status.status = "unreachable";
                    routingTable[payload.nodeID] = tmpRoute;
                }

                send this, ePeerDisconnected, (payload.nodeID,);
            }
        }

        // =====================================================================
        // MESSAGE ROUTING
        // =====================================================================

        on eRouteMessage do (payload: (message: RoutedMessage)) {
            tmpMessage = payload.message;

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
                deliveredMessages = deliveredMessages + (tmpMessage.id);
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

        on eQueryRoute do (payload: (destination: NodeID)) {
            tmpQueryRoute = default(seq[NodeID]);

            if (payload.destination in routingTable) {
                tmpEntry = routingTable[payload.destination];

                if (tmpEntry.status.status == "active") {
                    tmpQueryRoute += (0, localNode);
                    tmpQueryRoute += (1, tmpEntry.nextHop);
                    tmpQueryRoute += (2, payload.destination);
                }
            }

            send this, eRouteQueryResult, (tmpQueryRoute,);
        }

        // =====================================================================
        // ROUTING TABLE UPDATES
        // =====================================================================

        on eUpdateRoute do (payload: (routeEntry: RouteEntry)) {
            // Update routing table with new information
            tmpExistingMetric = 999999;

            if (payload.routeEntry.destination in routingTable) {
                tmpExistingMetric = routingTable[payload.routeEntry.destination].metric;
            }

            // Only update if better route
            if (payload.routeEntry.metric < tmpExistingMetric) {
                routingTable[payload.routeEntry.destination] = payload.routeEntry;
                send this, eRouteUpdated, (payload.routeEntry,);
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

            announce eBroadcastDelivered, (tmpMsgID, tmpReachedNodes);
            send this, eBroadcastDelivered, (tmpMsgID, tmpReachedNodes);
        }

        // =====================================================================
        // NETWORK PARTITIONING
        // =====================================================================

        on eNetworkPartitionDetected do (payload: (partitions: seq[seq[NodeID]])) {
            networkPartitioned = true;

            // Mark routes to other partitions as unreachable
            foreach (nodeID in keys(routingTable)) {
                tmpInOurPartition = false;

                // Check if node is in our partition (first partition)
                if (sizeof(payload.partitions) > 0) {
                    tmpOurPartition = payload.partitions[0];

                    tmpI = 0;
                    while (tmpI < sizeof(tmpOurPartition)) {
                        if (tmpOurPartition[tmpI].id == nodeID.id) {
                            tmpInOurPartition = true;
                            break;
                        }
                        tmpI = tmpI + 1;
                    }
                }

                if (!tmpInOurPartition) {
                    tmpRoute = routingTable[nodeID];
                    tmpRoute.status.status = "unreachable";
                    routingTable[nodeID] = tmpRoute;
                }
            }

            announce eMonitorViolation, ("NetworkTopology", "Network partition detected", default(map[string, any]));
        }

        on eNetworkHealed do (payload: (timestamp: Timestamp)) {
            networkPartitioned = false;

            // Re-enable all routes
            foreach (nodeID in keys(routingTable)) {
                if (nodeID in connectedPeers) {
                    tmpRoute = routingTable[nodeID];
                    tmpRoute.status.status = "active";
                    routingTable[nodeID] = tmpRoute;
                }
            }
        }

        // =====================================================================
        // PEER DISCOVERY (CONTINUOUS)
        // =====================================================================

        on eDiscoverPeers do {
            // Trigger peer discovery
            // In real system, would query DHT or bootstrap nodes
            announce ePeersDiscovered, (default(seq[PeerInfo]),);
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
