// p2p_interface.p - Abstract P State Machine for P2P Network Layer
// Models the P2P network interface as an abstract machine.
// This is an abstract interface representing external P2P communication.
// Based on: KNIRVGATEWAY/internal/oracle/p2p/manager.go

// =====================================================================
// P2P Interface - Abstract Machine
// This machine represents the P2P layer's interface without fully
// specifying its internal implementation. Used for compositional testing.
// =====================================================================

machine P2PInterface {
    // State variables
    var isRunning: bool;
    var isPaused: bool;
    var peers: map[string, PeerInfo];
    var peerCount: int;
    var inboundPeers: int;
    var outboundPeers: int;
    var listenAddr: string;
    var enableDHT: bool;
    var enableGossip: bool;

    // Start state
    start state Init {
        entry {
            isRunning = false;
            isPaused = false;
            peers = default(map[string, PeerInfo]);
            peerCount = 0;
            inboundPeers = 0;
            outboundPeers = 0;
            listenAddr = "/ip4/0.0.0.0/tcp/26656";
            enableDHT = true;
            enableGossip = true;

            announce eComponentStarted, (componentName = "P2PInterface");
        }

        on eStartP2P do {
            goto Running;
        }

        on eOracleStart do {
            goto Running;
        }
    }

    state Running {
        entry {
            isRunning = true;
            isPaused = false;
            announce eP2PStarted, ();
        }

        // Pause P2P networking
        on ePauseP2P do {
            goto Paused;
        }

        // Stop P2P networking
        on eStopP2P do {
            goto Stopped;
        }

        on eOracleStop do {
            goto Stopped;
        }

        // Broadcast a message to all peers
        on eBroadcastMessage do (payload: (messageType: string, data: seq[int])) {
            if (peerCount == 0) {
                announce eP2PError, (
                    operation = "Broadcast",
                    error = (code = 501, message = "No connected peers")
                );
                return;
            }

            // In abstract machine, we just announce success
            announce eMessageBroadcasted, (messageType = payload.messageType);
        }

        // Send message to specific peer
        on eSendToPeer do (payload: (peerID: string, messageType: string, data: seq[int])) {
            if (!(payload.peerID in peers)) {
                announce eP2PError, (
                    operation = "SendToPeer",
                    error = (code = 502, message = "Peer not found")
                );
                return;
            }

            announce eMessageSent, (
                peerID = payload.peerID,
                messageType = payload.messageType
            );
        }

        // Connect to a new peer
        on eConnectToPeer do (payload: (peerAddress: string)) {
            // Simulate peer connection
            var newPeerID: string;
            newPeerID = format("peer_{0}", peerCount);

            var peerInfo: PeerInfo;
            peerInfo = (
                peerID = newPeerID,
                address = payload.peerAddress,
                connectedAt = 0,
                isActive = true
            );

            peers[newPeerID] = peerInfo;
            peerCount = peerCount + 1;
            outboundPeers = outboundPeers + 1;

            announce ePeerConnected, (peerInfo = peerInfo);
        }

        // Disconnect from a peer
        on eDisconnectPeer do (payload: (peerID: string)) {
            if (!(payload.peerID in peers)) {
                return;
            }

            peers -= payload.peerID;
            peerCount = peerCount - 1;
            outboundPeers = outboundPeers - 1;

            announce ePeerDisconnected, (peerID = payload.peerID);
        }

        // Get P2P status
        on eGetP2PStatus do {
            var stats: P2PStats;
            stats = (
                peerCount = peerCount,
                inboundPeers = inboundPeers,
                outboundPeers = outboundPeers,
                isRunning = isRunning
            );

            announce eP2PStatusResponse, (stats = stats);
        }

        // List all connected peers
        on eListPeers do {
            var peerList: seq[PeerInfo];
            peerList = default(seq[PeerInfo]);

            foreach (id in keys(peers)) {
                peerList += (sizeof(peerList), peers[id]);
            }

            announce ePeersListResponse, (peers = peerList);
        }

        // Simulate receiving a message from a peer
        // This would be called by the test harness to simulate network activity
        on eMessageReceived do (payload: (fromPeer: string, messageType: string, data: seq[int])) {
            // Message received from network - forward to appropriate handler
            announce eMessageReceived, (
                fromPeer = payload.fromPeer,
                messageType = payload.messageType,
                data = payload.data
            );
        }
    }

    state Paused {
        entry {
            isPaused = true;
            announce eP2PPaused, ();
        }

        // Resume P2P networking
        on eResumeP2P do {
            goto Running;
        }

        // Stop while paused
        on eStopP2P do {
            goto Stopped;
        }

        on eOracleStop do {
            goto Stopped;
        }

        // Still respond to status queries while paused
        on eGetP2PStatus do {
            var stats: P2PStats;
            stats = (
                peerCount = peerCount,
                inboundPeers = inboundPeers,
                outboundPeers = outboundPeers,
                isRunning = false  // Running = false when paused
            );

            announce eP2PStatusResponse, (stats = stats);
        }

        on eListPeers do {
            var peerList: seq[PeerInfo];
            peerList = default(seq[PeerInfo]);
            foreach (id in keys(peers)) {
                peerList += (sizeof(peerList), peers[id]);
            }
            announce ePeersListResponse, (peers = peerList);
        }

        // Ignore network operations while paused
        ignore eBroadcastMessage, eSendToPeer, eConnectToPeer, eDisconnectPeer;
        ignore eMessageReceived;
    }

    state Stopped {
        entry {
            isRunning = false;
            isPaused = false;
            announce eP2PStopped, ();
            announce eComponentStopped, (componentName = "P2PInterface");
        }

        // Can restart
        on eStartP2P do {
            goto Running;
        }

        on eOracleStart do {
            goto Running;
        }

        // Still respond to status queries
        on eGetP2PStatus do {
            var stats: P2PStats;
            stats = (
                peerCount = peerCount,
                inboundPeers = inboundPeers,
                outboundPeers = outboundPeers,
                isRunning = false
            );
            announce eP2PStatusResponse, (stats = stats);
        }

        on eListPeers do {
            var peerList: seq[PeerInfo];
            peerList = default(seq[PeerInfo]);
            foreach (id in keys(peers)) {
                peerList += (sizeof(peerList), peers[id]);
            }
            announce ePeersListResponse, (peers = peerList);
        }

        ignore eBroadcastMessage, eSendToPeer, eConnectToPeer, eDisconnectPeer;
        ignore ePauseP2P, eResumeP2P;
        ignore eMessageReceived;
    }
}

// =====================================================================
// Abstract P2P Specification
// This specification defines the expected behavior of the P2P layer
// for compositional testing.
// =====================================================================

spec P2PSpecification observes eBroadcastMessage, eMessageBroadcasted, eP2PError {
    // Specification: Every broadcast should either succeed or report an error
    start state WaitForBroadcast {
        on eBroadcastMessage goto ExpectResponse;
    }

    hot state ExpectResponse {
        on eMessageBroadcasted goto WaitForBroadcast;
        on eP2PError goto WaitForBroadcast;
    }
}
