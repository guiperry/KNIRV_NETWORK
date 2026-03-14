// execution_sandbox.p - Execution Sandbox State Machine for KNIRVSERVER
// Models isolated execution environments for secure code/model validation
// Based on: KNIRVSERVER sandbox implementation

// Sandbox instance tracking - defined at file level
type SandboxInstance = (
    id: UUID,
    config: SandboxConfig,
    sandboxState: string,  // "created", "running", "completed", "destroyed"
    createdAt: Timestamp,
    resourcesUsed: ResourceUsage
);

// Execution state tracking - defined at file level
type ExecutionState = (
    sandboxID: UUID,
    startTime: Timestamp,
    resourcesConsumed: ResourceUsage,
    outputBuffer: seq[int]
);

machine ExecutionSandboxMachine {
    // State variables - Sandbox instances
    var sandboxes: map[UUID, SandboxInstance];
    var sandboxConfigs: map[UUID, SandboxConfig];
    var runningExecutions: map[UUID, ExecutionState];

    // Configuration
    var defaultConfig: SandboxConfig;
    var maxConcurrentSandboxes: int;
    var sandboxTimeout: int;

    // Counters
    var sandboxCounter: int;
    var totalExecutions: int;
    var successfulExecutions: int;
    var failedExecutions: int;
    var resourceViolations: int;

    // Temp variables for event handlers
    var tmpActiveSandboxes: int;
    var tmpSandboxID: UUID;
    var tmpNewSandbox: SandboxInstance;
    var tmpNewSandboxID: UUID;
    var tmpSandbox: SandboxInstance;
    var tmpExecState: ExecutionState;
    var tmpConfig: SandboxConfig;
    var tmpResourcesUsed: ResourceUsage;
    var tmpViolated: bool;
    var tmpViolationReason: string;
    var tmpOutput: map[string, any];
    var tmpCount: int;
    var allSandboxKeys: seq[UUID];
    var tmpI: int;
    var tmpSandboxCreatedPayload: (sandboxID: UUID);
    var tmpSandboxDestroyedPayload: (sandboxID: UUID);

    start state Init {
        entry {
            sandboxes = default(map[UUID, SandboxInstance]);
            sandboxConfigs = default(map[UUID, SandboxConfig]);
            runningExecutions = default(map[UUID, ExecutionState]);

            // Default sandbox configuration
            defaultConfig.maxCPU = 1000;      // 1000ms CPU time
            defaultConfig.maxMemory = 536870912;  // 512MB
            defaultConfig.maxStorage = 104857600;  // 100MB
            defaultConfig.maxNetwork = 10485760;   // 10MB
            defaultConfig.timeout = 60000;    // 60 seconds
            defaultConfig.allowedSyscalls = default(seq[string]);
            defaultConfig.isolationLevel = "strict";

            maxConcurrentSandboxes = 10;
            sandboxTimeout = 60000;

            sandboxCounter = 0;
            totalExecutions = 0;
            successfulExecutions = 0;
            failedExecutions = 0;
            resourceViolations = 0;
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // =====================================================================
        // SANDBOX CREATION
        // =====================================================================

        on eCreateSandbox do (payload: (config: SandboxConfig)) {
            // Check concurrent sandbox limit
            tmpActiveSandboxes = 0;
            allSandboxKeys = keys(sandboxes); // Assign to the machine-level variable
            tmpI = 0;
            while (tmpI < sizeof(allSandboxKeys)) {
                tmpSandboxID = allSandboxKeys[tmpI]; // Get the current element
                if (sandboxes[tmpSandboxID].sandboxState == "created" || sandboxes[tmpSandboxID].sandboxState == "running") {
                    tmpActiveSandboxes = tmpActiveSandboxes + 1;
                }
                tmpI = tmpI + 1; // Increment loop counter
            }

            if (tmpActiveSandboxes >= maxConcurrentSandboxes) {
                send this, eSandboxCreationFailed, "Maximum concurrent sandboxes reached";
                return;
            }

            // Validate config
            if (payload.config.maxCPU <= 0 || payload.config.maxMemory <= 0) {
                send this, eSandboxCreationFailed, "Invalid sandbox configuration";
                return;
            }

            // Create sandbox
            sandboxCounter = sandboxCounter + 1;
            tmpSandboxID.value = format("sandbox_{0}", sandboxCounter);

            tmpNewSandbox.id = tmpSandboxID;
            tmpNewSandbox.config = payload.config;
            tmpNewSandbox.sandboxState = "created";
            tmpNewSandbox.createdAt.milliseconds = 0;
            tmpNewSandbox.resourcesUsed.cpuMillis = 0;
            tmpNewSandbox.resourcesUsed.memoryBytes = 0;
            tmpNewSandbox.resourcesUsed.storageBytes = 0;
            tmpNewSandbox.resourcesUsed.networkBytes = 0;

            sandboxes[tmpSandboxID] = tmpNewSandbox;
            sandboxConfigs[tmpSandboxID] = payload.config;

            tmpSandboxCreatedPayload.sandboxID = tmpSandboxID;
            announce eSandboxCreated, tmpSandboxCreatedPayload;
            send this, eSandboxCreated, tmpSandboxCreatedPayload;
        }

        // =====================================================================
        // CODE EXECUTION
        // =====================================================================

        on eExecuteInSandbox do (payload: (
            sandboxID: UUID,
            code: seq[int],
            inputs: map[string, any]
        )) {
            // Check sandbox exists
            if (!(payload.sandboxID in sandboxes)) {
                // Create on-demand sandbox with default config
                sandboxCounter = sandboxCounter + 1;
                tmpNewSandboxID = payload.sandboxID;

                tmpNewSandbox.id = tmpNewSandboxID;
                tmpNewSandbox.config = defaultConfig;
                tmpNewSandbox.sandboxState = "created";
                tmpNewSandbox.createdAt.milliseconds = 0;
                tmpNewSandbox.resourcesUsed.cpuMillis = 0;
                tmpNewSandbox.resourcesUsed.memoryBytes = 0;
                tmpNewSandbox.resourcesUsed.storageBytes = 0;
                tmpNewSandbox.resourcesUsed.networkBytes = 0;

                sandboxes[tmpNewSandboxID] = tmpNewSandbox;
                sandboxConfigs[tmpNewSandboxID] = defaultConfig;
            }

            tmpSandbox = sandboxes[payload.sandboxID];

            // Check sandbox state
            if (tmpSandbox.sandboxState != "created") {
                send this, eSandboxExecutionFailed, (sandboxID = payload.sandboxID, reason = "Sandbox not in created state");
                return;
            }

            // Start execution
            tmpSandbox.sandboxState = "running";
            sandboxes[payload.sandboxID] = tmpSandbox;

            tmpExecState.sandboxID = payload.sandboxID;
            tmpExecState.startTime.milliseconds = 0;
            tmpExecState.resourcesConsumed.cpuMillis = 0;
            tmpExecState.resourcesConsumed.memoryBytes = 0;
            tmpExecState.resourcesConsumed.storageBytes = 0;
            tmpExecState.resourcesConsumed.networkBytes = 0;
            tmpExecState.outputBuffer = default(seq[int]);
            runningExecutions[payload.sandboxID] = tmpExecState;

            totalExecutions = totalExecutions + 1;

            // Simulate execution (in real system, would actually execute)
            goto Executing;
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            // Destroy all running sandboxes
            allSandboxKeys = keys(sandboxes); // Assign to the machine-level variable
            tmpI = 0;
            while (tmpI < sizeof(allSandboxKeys)) {
                tmpSandboxID = allSandboxKeys[tmpI]; // Get the current element
                if (sandboxes[tmpSandboxID].sandboxState == "running") {
                    tmpSandbox = sandboxes[tmpSandboxID];
                    tmpSandbox.sandboxState = "destroyed";
                    sandboxes[tmpSandboxID] = tmpSandbox;
                }
                tmpI = tmpI + 1; // Increment loop counter
            }
            goto Halted;
        }
    }

    state Executing {
        entry {
            // Simulate execution for all running sandboxes
            allSandboxKeys = keys(runningExecutions); // Assign to the machine-level variable
            tmpI = 0;
            while (tmpI < sizeof(allSandboxKeys)) {
                tmpSandboxID = allSandboxKeys[tmpI]; // Get the current element
                tmpExecState = runningExecutions[tmpSandboxID];

                tmpConfig = sandboxConfigs[tmpSandboxID];

                // Simulate resource usage
                tmpResourcesUsed.cpuMillis = 100;      // Simulated CPU usage
                tmpResourcesUsed.memoryBytes = 10485760;  // 10MB
                tmpResourcesUsed.storageBytes = 1048576;  // 1MB
                tmpResourcesUsed.networkBytes = 0;

                // Check resource limits
                tmpViolated = false;
                tmpViolationReason = "";

                if (tmpResourcesUsed.cpuMillis > tmpConfig.maxCPU) {
                    tmpViolated = true;
                    tmpViolationReason = "CPU limit exceeded";
                }

                if (tmpResourcesUsed.memoryBytes > tmpConfig.maxMemory) {
                    tmpViolated = true;
                    tmpViolationReason = "Memory limit exceeded";
                }

                if (tmpResourcesUsed.storageBytes > tmpConfig.maxStorage) {
                    tmpViolated = true;
                    tmpViolationReason = "Storage limit exceeded";
                }

                if (tmpResourcesUsed.networkBytes > tmpConfig.maxNetwork) {
                    tmpViolated = true;
                    tmpViolationReason = "Network limit exceeded";
                }

                if (tmpViolated) {
                    // Resource violation
                    resourceViolations = resourceViolations + 1;
                    failedExecutions = failedExecutions + 1;

                    announce eResourceLimitExceeded, (sandboxID = tmpSandboxID, resourceType = tmpViolationReason, limit = tmpConfig.maxCPU, actual = tmpResourcesUsed.cpuMillis);

                    announce eSandboxExecutionFailed, (sandboxID = tmpSandboxID, reason = tmpViolationReason);
                    send this, eSandboxExecutionFailed, (sandboxID = tmpSandboxID, reason = tmpViolationReason);
                } else {
                    // Successful execution
                    successfulExecutions = successfulExecutions + 1;

                    tmpOutput = default(map[string, any]);
                    tmpOutput["status"] = "success";
                    tmpOutput["execution_time"] = tmpResourcesUsed.cpuMillis;

                    announce eSandboxExecutionResult, (sandboxID = tmpSandboxID, output = tmpOutput, resourcesUsed = tmpResourcesUsed);
                    send this, eSandboxExecutionResult, (sandboxID = tmpSandboxID, output = tmpOutput, resourcesUsed = tmpResourcesUsed);
                }

                // Update sandbox state
                tmpSandbox = sandboxes[tmpSandboxID];
                tmpSandbox.sandboxState = "completed";
                tmpSandbox.resourcesUsed = tmpResourcesUsed;
                sandboxes[tmpSandboxID] = tmpSandbox;

                // Remove from running
                runningExecutions -= tmpSandboxID;
                tmpI = tmpI + 1; // Increment loop counter
            }

            goto Active;
        }
    }

    state Halted {
        on eEmergencyResume do {
            goto Active;
        }
    }

    // =====================================================================
    // SANDBOX DESTRUCTION
    // =====================================================================

    // This runs in Active state
    fun DestroySandbox(sandboxID: UUID) {
        if (sandboxID in sandboxes) {
            tmpSandbox = sandboxes[sandboxID];
            tmpSandbox.sandboxState = "destroyed";
            sandboxes[sandboxID] = tmpSandbox;

            // Clean up execution state if any
            if (sandboxID in runningExecutions) {
                runningExecutions -= sandboxID;
            }

            tmpSandboxDestroyedPayload.sandboxID = sandboxID;
            announce eSandboxDestroyed, tmpSandboxDestroyedPayload;
        }
    }

    // =====================================================================
    // HELPER FUNCTIONS
    // =====================================================================

    fun GetActiveSandboxCount(): int {
        tmpCount = 0;
        allSandboxKeys = keys(sandboxes); // Assign to the machine-level variable
        tmpI = 0;
        while (tmpI < sizeof(allSandboxKeys)) {
            tmpSandboxID = allSandboxKeys[tmpI]; // Get the current element
            if (sandboxes[tmpSandboxID].sandboxState == "created" || sandboxes[tmpSandboxID].sandboxState == "running") {
                tmpCount = tmpCount + 1;
            }
            tmpI = tmpI + 1; // Increment loop counter
        }
        return tmpCount;
    }

    fun GetTotalExecutions(): int {
        return totalExecutions;
    }

    fun GetSuccessRate(): int {
        if (totalExecutions == 0) {
            return 100;
        }
        return (successfulExecutions * 100) / totalExecutions;
    }

    fun GetResourceViolationCount(): int {
        return resourceViolations;
    }
}
