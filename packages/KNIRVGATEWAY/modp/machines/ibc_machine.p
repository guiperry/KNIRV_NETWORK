// ibc_machine.p - P State Machine for IBC Handler
// Models Inter-Blockchain Communication (IBC) protocol handling.
// Based on: KNIRVGATEWAY/internal/oracle/ibc/handler.go

machine IBCMachine {
    // State variables
    var channels: map[string, IBCChannel];
    var connections: map[string, IBCConnection];
    var clients: map[string, IBCClient];
    var chainChannels: map[ChainID, string];  // Maps chain ID to channel ID
    var sequenceNumber: int;
    var pendingPackets: seq[IBCPacket];
    var isRunning: bool;

    // Start state
    start state Init {
        entry {
            channels = default(map[string, IBCChannel]);
            connections = default(map[string, IBCConnection]);
            clients = default(map[string, IBCClient]);
            chainChannels = default(map[ChainID, string]);
            sequenceNumber = 0;
            pendingPackets = default(seq[IBCPacket]);
            isRunning = false;

            announce eComponentStarted, (componentName = "IBCMachine");
        }

        on eOracleStart do {
            goto Active;
        }
    }

    state Active {
        entry {
            isRunning = true;
        }

        // Register a new IBC channel
        on eRegisterChannel do (payload: (
            channelID: string,
            connectionID: string,
            portID: string,
            chainID: ChainID,
            ordering: ChannelOrdering
        )) {
            if (payload.channelID in channels) {
                announce eIBCOperationFailed, (
                    operation = "RegisterChannel",
                    error = (code = 401, message = "Channel already exists")
                );
                return;
            }

            var channel: IBCChannel;
            channel = (
                channelID = payload.channelID,
                connectionID = payload.connectionID,
                portID = payload.portID,
                state = (stateName = "Init"),
                version = "ics20-1",
                ordering = payload.ordering,
                chainID = payload.chainID,
                createdAt = 0
            );

            channels[payload.channelID] = channel;
            chainChannels[payload.chainID] = payload.channelID;

            announce eChannelRegistered, (channel = channel);
        }

        // Register a new IBC connection
        on eRegisterConnection do (payload: (
            connectionID: string,
            clientID: string,
            endpoint: string,
            chainID: ChainID
        )) {
            if (payload.connectionID in connections) {
                announce eIBCOperationFailed, (
                    operation = "RegisterConnection",
                    error = (code = 402, message = "Connection already exists")
                );
                return;
            }

            var connection: IBCConnection;
            connection = (
                connectionID = payload.connectionID,
                clientID = payload.clientID,
                state = (stateName = "Init"),
                chainID = payload.chainID,
                endpoint = payload.endpoint,
                createdAt = 0
            );

            connections[payload.connectionID] = connection;

            announce eConnectionRegistered, (connection = connection);
        }

        // Register a new light client
        on eRegisterClient do (payload: (
            clientID: string,
            clientType: string,
            chainID: ChainID,
            trustingPeriod: int
        )) {
            if (payload.clientID in clients) {
                announce eIBCOperationFailed, (
                    operation = "RegisterClient",
                    error = (code = 403, message = "Client already exists")
                );
                return;
            }

            var client: IBCClient;
            client = (
                clientID = payload.clientID,
                clientType = payload.clientType,
                chainID = payload.chainID,
                state = (stateName = "Active"),
                latestHeight = 0,
                trustingPeriod = payload.trustingPeriod,
                createdAt = 0
            );

            clients[payload.clientID] = client;

            announce eClientRegistered, (client = client);
        }

        // Open a channel
        on eOpenChannel do (payload: (channelID: string)) {
            if (!(payload.channelID in channels)) {
                announce eIBCOperationFailed, (
                    operation = "OpenChannel",
                    error = ChannelNotFound()
                );
                return;
            }

            var channel: IBCChannel;
            channel = channels[payload.channelID];
            channel.state = (stateName = "Open");
            channels[payload.channelID] = channel;

            announce eChannelOpened, (channelID = payload.channelID);
        }

        // Open a connection
        on eOpenConnection do (payload: (connectionID: string)) {
            if (!(payload.connectionID in connections)) {
                announce eIBCOperationFailed, (
                    operation = "OpenConnection",
                    error = (code = 404, message = "Connection not found")
                );
                return;
            }

            var connection: IBCConnection;
            connection = connections[payload.connectionID];
            connection.state = (stateName = "Open");
            connections[payload.connectionID] = connection;

            announce eConnectionOpened, (connectionID = payload.connectionID);
        }

        // Send an IBC packet
        on eSendPacket do (payload: (packet: IBCPacket)) {
            // Assign sequence number
            sequenceNumber = sequenceNumber + 1;
            var packet: IBCPacket;
            packet = payload.packet;
            packet.sequence = sequenceNumber;

            // Validate destination channel exists
            if (!(packet.destChainID in chainChannels)) {
                announce eIBCOperationFailed, (
                    operation = "SendPacket",
                    error = (code = 405, message = "No channel for destination chain")
                );
                return;
            }

            // Add to pending packets
            pendingPackets += (sizeof(pendingPackets), packet);

            announce ePacketSent, (packet = packet);
        }

        // Receive an IBC packet
        on eReceivePacket do (payload: (packet: IBCPacket)) {
            // Validate source channel
            var sourceChannelID: string;
            if (payload.packet.sourceChainID in chainChannels) {
                sourceChannelID = chainChannels[payload.packet.sourceChainID];
            } else {
                announce eIBCOperationFailed, (
                    operation = "ReceivePacket",
                    error = (code = 406, message = "Unknown source chain")
                );
                return;
            }

            // Validate channel is open
            if (sourceChannelID in channels) {
                var channel: IBCChannel;
                channel = channels[sourceChannelID];
                if (channel.state.stateName != "Open") {
                    announce eIBCOperationFailed, (
                        operation = "ReceivePacket",
                        error = (code = 407, message = "Channel not open")
                    );
                    return;
                }
            }

            announce ePacketReceived, (packet = payload.packet);
        }

        // Acknowledge packet receipt
        on eAcknowledgePacket do (payload: (sequence: int, channelID: string, success: bool)) {
            // Remove from pending packets
            var i: int;
            var newPending: seq[IBCPacket];
            newPending = default(seq[IBCPacket]);

            i = 0;
            while (i < sizeof(pendingPackets)) {
                if (pendingPackets[i].sequence != payload.sequence) {
                    newPending += (sizeof(newPending), pendingPackets[i]);
                }
                i = i + 1;
            }
            pendingPackets = newPending;

            announce ePacketAcknowledged, (
                sequence = payload.sequence,
                channelID = payload.channelID,
                success = payload.success
            );
        }

        // Query channel
        on eGetChannel do (payload: (channelID: string)) {
            if (!(payload.channelID in channels)) {
                announce eIBCOperationFailed, (
                    operation = "GetChannel",
                    error = ChannelNotFound()
                );
                return;
            }

            announce eChannelResponse, (channel = channels[payload.channelID]);
        }

        // List all channels
        on eListChannels do {
            var channelList: seq[IBCChannel];
            channelList = default(seq[IBCChannel]);

            foreach (id in keys(channels)) {
                channelList += (sizeof(channelList), channels[id]);
            }

            announce eChannelsListResponse, (channels = channelList);
        }

        // List all connections
        on eListConnections do {
            var connectionList: seq[IBCConnection];
            connectionList = default(seq[IBCConnection]);

            foreach (id in keys(connections)) {
                connectionList += (sizeof(connectionList), connections[id]);
            }

            announce eConnectionsListResponse, (connections = connectionList);
        }

        // Handle shutdown
        on eOracleStop do {
            goto Stopped;
        }
    }

    state Stopped {
        entry {
            isRunning = false;
            announce eComponentStopped, (componentName = "IBCMachine");
        }

        on eOracleStart do {
            goto Active;
        }

        // Still respond to queries
        on eGetChannel do (payload: (channelID: string)) {
            if (payload.channelID in channels) {
                announce eChannelResponse, (channel = channels[payload.channelID]);
            }
        }

        on eListChannels do {
            var channelList: seq[IBCChannel];
            channelList = default(seq[IBCChannel]);
            foreach (id in keys(channels)) {
                channelList += (sizeof(channelList), channels[id]);
            }
            announce eChannelsListResponse, (channels = channelList);
        }

        on eListConnections do {
            var connectionList: seq[IBCConnection];
            connectionList = default(seq[IBCConnection]);
            foreach (id in keys(connections)) {
                connectionList += (sizeof(connectionList), connections[id]);
            }
            announce eConnectionsListResponse, (connections = connectionList);
        }

        ignore eRegisterChannel, eRegisterConnection, eRegisterClient;
        ignore eOpenChannel, eOpenConnection;
        ignore eSendPacket, eReceivePacket, eAcknowledgePacket;
    }
}
