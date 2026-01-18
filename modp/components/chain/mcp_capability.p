// mcp_capability.p - MCP Capability State Machine for KNIRVCHAIN
// Models the Model Context Protocol capability registration and invocation
// Based on: KNIRVCHAIN MCP capabilities implementation

machine MCPCapabilityMachine {
    // State variables
    var capabilities: map[UUID, MCPCapability];
    var capabilitiesByName: map[string, UUID];
    var capabilitiesByOwner: map[Address, seq[UUID]];
    var capabilityCounter: int;

    // Usage tracking
    var usageCount: map[UUID, int];
    var usageByInvoker: map[Address, map[UUID, int]];

    // Configuration
    var registrationFee: BigInt;
    var invocationFee: BigInt;
    var maxCapabilitiesPerOwner: int;

    // Permission levels
    var permissionLevels: map[string, int];

    // Temp variables for event handlers
    var tmpOwnerCapabilities: seq[UUID];
    var tmpCapabilityID: UUID;
    var tmpNewCapability: MCPCapability;
    var tmpCapability: MCPCapability;
    var tmpInvokerUsage: map[UUID, int];
    var tmpResult: map[string, any];
    var tmpMintedCapability: MCPCapability;
    var tmpStatus: MCPStatus;
    var tmpCapResult: MCPCapabilityResultData;

    start state Init {
        entry {
            capabilities = default(map[UUID, MCPCapability]);
            capabilitiesByName = default(map[string, UUID]);
            capabilitiesByOwner = default(map[Address, seq[UUID]]);
            capabilityCounter = 0;

            usageCount = default(map[UUID, int]);
            usageByInvoker = default(map[Address, map[UUID, int]]);

            registrationFee.value = 5000000;  // 5 NRN
            registrationFee.isNegative = false;
            invocationFee.value = 100000;  // 0.1 NRN
            invocationFee.isNegative = false;
            maxCapabilitiesPerOwner = 50;

            // Permission levels
            permissionLevels = default(map[string, int]);
            permissionLevels["read"] = 1;
            permissionLevels["write"] = 2;
            permissionLevels["execute"] = 3;
            permissionLevels["admin"] = 4;
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // MCP capability registration
        on eRegisterMCPCapability do (payload: (
            name: string,
            version: string,
            serverPointer: string,
            owner: Address,
            permissions: seq[string]
        )) {
            // Check if name already exists
            if (payload.name in capabilitiesByName) {
                send this, eMCPCapabilityFailed, "Capability name already registered";
                return;
            }

            // Check owner limit
            if (payload.owner in capabilitiesByOwner) {
                tmpOwnerCapabilities = capabilitiesByOwner[payload.owner];
                if (sizeof(tmpOwnerCapabilities) >= maxCapabilitiesPerOwner) {
                    send this, eMCPCapabilityFailed, "Owner has reached maximum capability limit";
                    return;
                }
            } else {
                tmpOwnerCapabilities = default(seq[UUID]);
            }

            // Validate server pointer
            if (payload.serverPointer == "") {
                send this, eMCPCapabilityFailed, "Invalid MCP server pointer";
                return;
            }

            // Create capability ID
            capabilityCounter = capabilityCounter + 1;
            tmpCapabilityID.value = format("mcp_{0}", capabilityCounter);

            // Create capability
            tmpStatus.status = "available";
            tmpNewCapability.id = tmpCapabilityID;
            tmpNewCapability.name = payload.name;
            tmpNewCapability.version = payload.version;
            tmpNewCapability.serverPointer = payload.serverPointer;
            tmpNewCapability.owner = payload.owner;
            tmpNewCapability.permissions = payload.permissions;
            tmpNewCapability.status = tmpStatus;

            // Store capability
            capabilities[tmpCapabilityID] = tmpNewCapability;
            capabilitiesByName[payload.name] = tmpCapabilityID;
            usageCount[tmpCapabilityID] = 0;

            // Add to owner's capabilities
            tmpOwnerCapabilities += (sizeof(tmpOwnerCapabilities), tmpCapabilityID);
            capabilitiesByOwner[payload.owner] = tmpOwnerCapabilities;

            announce eMCPCapabilityRegistered, tmpNewCapability;
            send this, eMCPCapabilityRegistered, tmpNewCapability;
        }

        // MCP capability invocation
        on eInvokeMCPCapability do (payload: (
            capabilityID: UUID,
            caller: Address,
            params: map[string, any]
        )) {
            // Check capability exists
            if (!(payload.capabilityID in capabilities)) {
                tmpResult = default(map[string, any]);
                tmpCapResult.capabilityID = payload.capabilityID;
                tmpCapResult.result = tmpResult;
                tmpCapResult.success = false;
                send this, eMCPCapabilityResult, tmpCapResult;
                return;
            }

            tmpCapability = capabilities[payload.capabilityID];

            // Check capability is available
            if (tmpCapability.status.status != "available") {
                tmpResult = default(map[string, any]);
                tmpCapResult.capabilityID = payload.capabilityID;
                tmpCapResult.result = tmpResult;
                tmpCapResult.success = false;
                send this, eMCPCapabilityResult, tmpCapResult;
                return;
            }

            // Check permissions (simplified)
            // In real implementation, verify caller has required permissions

            // Update usage count
            usageCount[payload.capabilityID] = usageCount[payload.capabilityID] + 1;

            // Track usage by invoker
            if (payload.caller in usageByInvoker) {
                tmpInvokerUsage = usageByInvoker[payload.caller];
            } else {
                tmpInvokerUsage = default(map[UUID, int]);
            }

            if (payload.capabilityID in tmpInvokerUsage) {
                tmpInvokerUsage[payload.capabilityID] = tmpInvokerUsage[payload.capabilityID] + 1;
            } else {
                tmpInvokerUsage[payload.capabilityID] = 1;
            }
            usageByInvoker[payload.caller] = tmpInvokerUsage;

            // Simulate successful invocation
            tmpResult = default(map[string, any]);
            tmpResult["status"] = "success";
            tmpResult["capability"] = tmpCapability.name;

            tmpCapResult.capabilityID = payload.capabilityID;
            tmpCapResult.result = tmpResult;
            tmpCapResult.success = true;
            send this, eMCPCapabilityResult, tmpCapResult;
        }

        // Handle capability node minted from context
        on eCapabilityNodeMinted do (capabilityNode: CapabilityNode) {
            // Create an MCP capability from the minted capability node
            capabilityCounter = capabilityCounter + 1;
            tmpCapabilityID.value = format("mcp_minted_{0}", capabilityCounter);

            tmpStatus.status = "available";
            tmpMintedCapability.id = tmpCapabilityID;
            tmpMintedCapability.name = format("MintedCapability_{0}", capabilityNode.id.value);
            tmpMintedCapability.version = "1.0.0";
            tmpMintedCapability.serverPointer = capabilityNode.mcpServer;
            tmpMintedCapability.owner = capabilityNode.creator;
            tmpMintedCapability.permissions = default(seq[string]);
            tmpMintedCapability.status = tmpStatus;

            capabilities[tmpCapabilityID] = tmpMintedCapability;
            usageCount[tmpCapabilityID] = 0;

            // Add to owner's capabilities
            if (capabilityNode.creator in capabilitiesByOwner) {
                tmpOwnerCapabilities = capabilitiesByOwner[capabilityNode.creator];
            } else {
                tmpOwnerCapabilities = default(seq[UUID]);
            }
            tmpOwnerCapabilities += (sizeof(tmpOwnerCapabilities), tmpCapabilityID);
            capabilitiesByOwner[capabilityNode.creator] = tmpOwnerCapabilities;

            announce eMCPCapabilityRegistered, tmpMintedCapability;
        }

        // Deprecate capability
        on eNetworkConfigUpdate do (payload: (config: NetworkConfig)) {
            // Handle network config updates that might affect capabilities
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
    fun GetCapabilityCount(): int {
        return sizeof(capabilities);
    }

    fun GetCapabilityUsage(capabilityID: UUID): int {
        if (capabilityID in usageCount) {
            return usageCount[capabilityID];
        }
        return 0;
    }

    fun IsCapabilityAvailable(capabilityID: UUID): bool {
        if (capabilityID in capabilities) {
            return capabilities[capabilityID].status.status == "available";
        }
        return false;
    }
}
