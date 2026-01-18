// ibc_machine.p - IBC Handler State Machine for KNIRVORACLE
// Models Inter-Blockchain Communication for cross-chain transfers
// Based on: KNIRVORACLE IBC module implementation

// ConnectionState type - defined at file level
type ConnectionState = (
    connectionID: string,
    clientID: string,
    counterpartyConnectionID: string,
    counterpartyClientID: string,
    connState: int  // 0=None, 1=Init, 2=TryOpen, 3=Open
);

machine IBCMachine {
    // State variables
    var channels: map[string, IBCChannel];
    var connections: map[string, ConnectionState];
    var pendingPackets: map[int, IBCPacket];
    var packetSequence: int;

    // Configuration
    var defaultTimeout: int;  // blocks

    // Temp variables
    var tempChannelID: string;
    var tempNewChannel: IBCChannel;
    var tempChannel: IBCChannel;
    var tempNewPacket: IBCPacket;
    var tempPacket: IBCPacket;
    var tempChannelState: ChannelState;
    var tempTimestamp: Timestamp;

    start state Init {
        entry {
            channels = default(map[string, IBCChannel]);
            connections = default(map[string, ConnectionState]);
            pendingPackets = default(map[int, IBCPacket]);
            packetSequence = 0;
            defaultTimeout = 1000;  // blocks
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // Channel operations
        on eIBCChannelOpen do (payload: (channel: IBCChannel)) {
            tempChannelID = payload.channel.channelID;

            // Check if channel already exists
            if (tempChannelID in channels) {
                send this, eIBCChannelOpenFailed, "Channel already exists";
                return;
            }

            // Validate channel parameters
            if (payload.channel.connectionID == "") {
                send this, eIBCChannelOpenFailed, "Invalid connection ID";
                return;
            }

            // Create channel in Init state
            tempNewChannel = payload.channel;
            tempChannelState.value = 1;  // Init
            tempNewChannel.channelState = tempChannelState;

            channels[tempChannelID] = tempNewChannel;

            // In real IBC, this would trigger handshake
            // For model, we'll auto-complete to Open state
            tempChannelState.value = 3;  // Open
            tempNewChannel.channelState = tempChannelState;
            channels[tempChannelID] = tempNewChannel;

            send this, eIBCChannelOpened, tempChannelID;
        }

        on eIBCChannelClose do (payload: (channelID: string)) {
            if (!(payload.channelID in channels)) {
                return;
            }

            tempChannel = channels[payload.channelID];
            tempChannelState.value = 4;  // Closed
            tempChannel.channelState = tempChannelState;
            channels[payload.channelID] = tempChannel;

            send this, eIBCChannelClosed, payload.channelID;
        }

        // Packet sending
        on eIBCSendPacket do (packet: IBCPacket) {
            // Verify channel is open
            if (!(packet.sourceChannel in channels)) {
                send this, eIBCPacketFailed, "Channel not found";
                return;
            }

            tempChannel = channels[packet.sourceChannel];

            if (tempChannel.channelState.value != 3) {  // Not Open
                send this, eIBCPacketFailed, "Channel not open";
                return;
            }

            // Assign sequence number
            packetSequence = packetSequence + 1;

            tempNewPacket = packet;
            tempNewPacket.sequence = packetSequence;

            // Store pending packet
            pendingPackets[packetSequence] = tempNewPacket;

            // Announce packet sent
            announce eIBCPacketSent, packetSequence;
            send this, eIBCPacketSent, packetSequence;
        }

        // Packet receiving (from counterparty chain)
        on eIBCReceivePacket do (payload: (packet: IBCPacket)) {
            // Verify channel exists and is open
            if (!(payload.packet.destChannel in channels)) {
                return;
            }

            tempChannel = channels[payload.packet.destChannel];

            if (tempChannel.channelState.value != 3) {
                return;
            }

            // Process packet based on port/data
            // In real implementation, this would dispatch to specific handlers

            send this, eIBCPacketReceived, payload.packet.sequence;

            // Send acknowledgment
            send this, eIBCPacketAcknowledged, payload.packet.sequence;
        }

        // Packet acknowledgment
        on eIBCPacketAcknowledged do (seqNum: int) {
            // Track acknowledged packets - just remove from pending
            if (seqNum in pendingPackets) {
                pendingPackets -= seqNum;
            }
        }

        // Packet timeout
        on eIBCPacketTimeout do (seqNum: int) {
            // Handle timeout - just remove from pending
            if (seqNum in pendingPackets) {
                pendingPackets -= seqNum;
                // Refund logic would go here
            }
        }

        // Cross-chain message handling
        on eSendCrossChainMessage do (payload: (message: CrossChainMessage)) {
            // Convert cross-chain message to IBC packet
            tempTimestamp.milliseconds = 0;

            tempPacket.sequence = 0;  // Will be assigned
            tempPacket.sourcePort = "transfer";
            tempPacket.sourceChannel = "channel-0";
            tempPacket.destPort = "transfer";
            tempPacket.destChannel = "channel-0";
            tempPacket.payload = payload.message.payload;
            tempPacket.timeoutHeight = defaultTimeout;
            tempPacket.timeoutTimestamp = tempTimestamp;

            send this, eIBCSendPacket, tempPacket;
            send this, eCrossChainMessageSent, payload.message.id;
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
    fun GetChannelState(channelID: string): int {
        if (channelID in channels) {
            return channels[channelID].channelState.value;
        }
        return 0;
    }

    fun GetPendingPacketCount(): int {
        return sizeof(pendingPackets);
    }

    fun IsChannelOpen(channelID: string): bool {
        if (channelID in channels) {
            return channels[channelID].channelState.value == 3;
        }
        return false;
    }
}
