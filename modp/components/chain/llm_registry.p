// llm_registry.p - LLM Registry State Machine for KNIRVCHAIN
// Models the registration and management of Large Language Models
// Based on: KNIRVCHAIN LLM registry implementation

machine LLMRegistryMachine {
    // State variables
    var registrations: map[UUID, LLMRegistration];
    var registrationsByProvider: map[Address, seq[UUID]];
    var registrationsByName: map[string, UUID];
    var registrationCounter: int;

    // Configuration
    var minStakingAmount: BigInt;
    var maxLLMsPerProvider: int;
    var slashingRate: int;  // basis points

    // References
    var tokenMachine: machine;
    var economicsMachine: machine;

    // Temp variables for event handlers
    var tmpProviderLLMs: seq[UUID];
    var tmpLLMID: UUID;
    var tmpNewRegistration: LLMRegistration;
    var tmpRegistration: LLMRegistration;
    var tmpSlashAmount: BigInt;
    var tmpStatus: LLMStatus;

    start state Init {
        entry {
            registrations = default(map[UUID, LLMRegistration]);
            registrationsByProvider = default(map[Address, seq[UUID]]);
            registrationsByName = default(map[string, UUID]);
            registrationCounter = 0;

            minStakingAmount.value = 1000000000;  // 1000 NRN
            minStakingAmount.isNegative = false;
            maxLLMsPerProvider = 10;
            slashingRate = 1000;  // 10%
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // LLM registration
        on eRegisterLLM do (payload: (
            name: string,
            provider: Address,
            modelType: string,
            apiEndpoint: string,
            capabilities: seq[string],
            stakingAmount: BigInt
        )) {
            // Check if name already exists
            if (payload.name in registrationsByName) {
                send this, eLLMRegistrationFailed, "LLM name already registered";
                return;
            }

            // Check minimum stake
            if (payload.stakingAmount.value < minStakingAmount.value) {
                send this, eLLMRegistrationFailed, "Insufficient staking amount";
                return;
            }

            // Check provider limit
            if (payload.provider in registrationsByProvider) {
                tmpProviderLLMs = registrationsByProvider[payload.provider];
                if (sizeof(tmpProviderLLMs) >= maxLLMsPerProvider) {
                    send this, eLLMRegistrationFailed, "Provider has reached maximum LLM limit";
                    return;
                }
            } else {
                tmpProviderLLMs = default(seq[UUID]);
            }

            // Validate API endpoint
            if (payload.apiEndpoint == "") {
                send this, eLLMRegistrationFailed, "Invalid API endpoint";
                return;
            }

            // Create registration ID
            registrationCounter = registrationCounter + 1;
            tmpLLMID.value = format("llm_{0}", registrationCounter);

            // Create registration
            tmpStatus.status = "registered";
            tmpNewRegistration.id = tmpLLMID;
            tmpNewRegistration.name = payload.name;
            tmpNewRegistration.provider = payload.provider;
            tmpNewRegistration.modelType = payload.modelType;
            tmpNewRegistration.apiEndpoint = payload.apiEndpoint;
            tmpNewRegistration.capabilities = payload.capabilities;
            tmpNewRegistration.stakingAmount = payload.stakingAmount;
            tmpNewRegistration.status = tmpStatus;

            // Store registration
            registrations[tmpLLMID] = tmpNewRegistration;
            registrationsByName[payload.name] = tmpLLMID;

            // Add to provider's registrations
            tmpProviderLLMs += (sizeof(tmpProviderLLMs), tmpLLMID);
            registrationsByProvider[payload.provider] = tmpProviderLLMs;

            // Activate after successful stake verification (simplified)
            tmpNewRegistration.status.status = "active";
            registrations[tmpLLMID] = tmpNewRegistration;

            announce eLLMRegistered, tmpNewRegistration;
            send this, eLLMRegistered, tmpNewRegistration;
        }

        // LLM suspension
        on eSuspendLLM do (payload: (llmID: UUID, reason: string)) {
            if (!(payload.llmID in registrations)) {
                return;
            }

            tmpRegistration = registrations[payload.llmID];

            if (tmpRegistration.status.status != "active") {
                return;
            }

            tmpRegistration.status.status = "suspended";
            registrations[payload.llmID] = tmpRegistration;

            // Apply slashing if suspended for malicious behavior
            if (payload.reason == "malicious") {
                tmpSlashAmount.value = (tmpRegistration.stakingAmount.value * slashingRate) / 10000;
                tmpSlashAmount.isNegative = false;

                // Slash would be sent to economics machine
            }

            send this, eLLMSuspended, payload.llmID;
        }

        // LLM reactivation
        on eReactivateLLM do (payload: (llmID: UUID)) {
            if (!(payload.llmID in registrations)) {
                return;
            }

            tmpRegistration = registrations[payload.llmID];

            if (tmpRegistration.status.status != "suspended") {
                return;
            }

            tmpRegistration.status.status = "active";
            registrations[payload.llmID] = tmpRegistration;

            send this, eLLMReactivated, payload.llmID;
        }

        // Handle errors from registered LLMs
        on eSubmitError do (payload: (
            errorType: string,
            errorMessage: string,
            context: map[string, any],
            sourceModel: UUID
        )) {
            // Record that this LLM produced an error
            // This could affect reputation/slashing
            if (payload.sourceModel in registrations) {
                tmpRegistration = registrations[payload.sourceModel];

                // In real implementation, track error rates
                // and potentially slash or suspend high-error LLMs
            }

            // Forward to knowledge graph
            announce eSubmitError, payload;
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
    fun GetLLMCount(): int {
        return sizeof(registrations);
    }

    fun GetActiveLLMCount(): int {
        var count: int;
        var id: UUID;
        count = 0;
        foreach (id in keys(registrations)) {
            if (registrations[id].status.status == "active") {
                count = count + 1;
            }
        }
        return count;
    }

    fun IsLLMActive(llmID: UUID): bool {
        if (llmID in registrations) {
            return registrations[llmID].status.status == "active";
        }
        return false;
    }

    fun GetLLMCapabilities(llmID: UUID): seq[string] {
        if (llmID in registrations) {
            return registrations[llmID].capabilities;
        }
        return default(seq[string]);
    }
}
