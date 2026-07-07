// validation_machine.p - Validation Environment State Machine for KNIRVSERVER
// Models the Distributed Validation Environment (DVE) for task validation
// Based on: KNIRVSERVER validation implementation

// Validator state tracking - defined at file level
type ValidatorState = (
    nodeID: NodeID,
    available: bool,
    currentTask: UUID,
    completedCount: int,
    failedCount: int,
    averageTime: int
);

machine ValidationMachine {
    // State variables - Task management
    var taskQueue: seq[ValidationTask];
    var runningTasks: map[UUID, ValidationTask];
    var completedTasks: map[UUID, ValidationResult];
    var failedTasks: map[UUID, string];  // taskID -> reason

    // State variables - Validator management
    var validators: map[NodeID, ValidatorState];
    var validatorQueue: seq[NodeID];

    // Configuration
    var maxQueueSize: int;
    var maxConcurrentTasks: int;
    var defaultTimeout: int;
    var maxRetries: int;

    // Counters
    var taskCounter: int;
    var totalTasksCompleted: int;
    var totalTasksFailed: int;

    // Sandbox reference
    var sandboxMachine: machine;

    // Temp variables for event handlers
    var tmpTaskID: UUID;
    var tmpNewTask: ValidationTask;
    var tmpFound: bool;
    var tmpTaskIdx: int;
    var tmpI: int;
    var tmpValidatorState: ValidatorState;
    var tmpTask: ValidationTask;
    var tmpNewQueue: seq[ValidationTask];
    var tmpValidatorID: NodeID;
    var tmpResult: ValidationResult;
    var tmpInserted: bool;
    var tmpAvailableValidator: NodeID;
    var tmpNextTask: ValidationTask;
    var tmpTotal: int;
    var validationResultPayload: (result: ValidationResult);
    var tmpValidationRejectedReason: ValidationRejectedReason;
    var tmpValidationTaskPayload: (task: ValidationTask);
    var tmpValidationStartedPayload: (taskID: UUID);
    var tmpValidationTimeoutPayload: (taskID: UUID);
    var allValidatorKeys: seq[NodeID];

    start state Init {
        entry {
            taskQueue = default(seq[ValidationTask]);
            runningTasks = default(map[UUID, ValidationTask]);
            completedTasks = default(map[UUID, ValidationResult]);
            failedTasks = default(map[UUID, string]);

            validators = default(map[NodeID, ValidatorState]);
            validatorQueue = default(seq[NodeID]);

            maxQueueSize = 1000;
            maxConcurrentTasks = 10;
            defaultTimeout = 300000;  // 5 minutes
            maxRetries = 3;

            taskCounter = 0;
            totalTasksCompleted = 0;
            totalTasksFailed = 0;
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
        // TASK SUBMISSION
        // =====================================================================

        on eSubmitValidationTask do (payload: (
            taskType: ValidationTaskType,
            taskPayload: map[string, any],
            submitter: Address,
            priority: int,
            deadline: Timestamp,
            caller: machine
        )) {
            // Check queue capacity
            if (sizeof(taskQueue) >= maxQueueSize) {
                tmpValidationRejectedReason.reason = "Task queue is full";
                send payload.caller, eValidationTaskRejected, tmpValidationRejectedReason;
                return;
            }

            // Create task
            taskCounter = taskCounter + 1;
            tmpTaskID.value = format("task_{0}", taskCounter);

            tmpNewTask.id = tmpTaskID;
            tmpNewTask.taskType = payload.taskType;
            tmpNewTask.payload = payload.taskPayload;
            tmpNewTask.submitter = payload.submitter;
            tmpNewTask.priority = payload.priority;
            tmpNewTask.deadline = payload.deadline;
            tmpNewTask.status.status = "queued";

            // Add to queue (priority-sorted insertion)
            InsertTaskByPriority(tmpNewTask);

            tmpValidationTaskPayload.task = tmpNewTask;
            announce eValidationTaskQueued, tmpValidationTaskPayload;
            send payload.caller, eValidationTaskQueued, tmpValidationTaskPayload;

            // Try to start task immediately if validator available
            TryStartNextTask();
        }

        // =====================================================================
        // TASK EXECUTION
        // =====================================================================

        on eStartValidation do (payload: (taskID: UUID, validator: NodeID)) {
            // Check task exists and is queued
            tmpFound = false;
            tmpTaskIdx = 0;

            tmpI = 0;
            while (tmpI < sizeof(taskQueue)) {
                if (taskQueue[tmpI].id.value == payload.taskID.value) {
                    tmpFound = true;
                    tmpTaskIdx = tmpI;
                    break;
                }
                tmpI = tmpI + 1;
            }

            if (!tmpFound) {
                return;
            }

            // Check validator is available
            if (payload.validator in validators) {
                tmpValidatorState = validators[payload.validator];
                if (!tmpValidatorState.available) {
                    return;
                }
            }

            // Move task from queue to running
            tmpTask = taskQueue[tmpTaskIdx];
            tmpTask.status.status = "running";

            // Remove from queue
            tmpNewQueue = default(seq[ValidationTask]);
            tmpI = 0;
            while (tmpI < sizeof(taskQueue)) {
                if (tmpI != tmpTaskIdx) {
                    tmpNewQueue += (sizeof(tmpNewQueue), taskQueue[tmpI]);
                }
                tmpI = tmpI + 1;
            }
            taskQueue = tmpNewQueue;

            // Add to running
            runningTasks[tmpTask.id] = tmpTask;

            // Update validator state
            if (payload.validator in validators) {
                tmpValidatorState = validators[payload.validator];
            } else {
                tmpValidatorState.nodeID = payload.validator;
                tmpValidatorState.available = true;
                tmpValidatorState.currentTask.value = "";
                tmpValidatorState.completedCount = 0;
                tmpValidatorState.failedCount = 0;
                tmpValidatorState.averageTime = 0;
            }
            tmpValidatorState.available = false;
            tmpValidatorState.currentTask = tmpTask.id;
            validators[payload.validator] = tmpValidatorState;

            tmpValidationStartedPayload.taskID = tmpTask.id;
            announce eValidationStarted, tmpValidationStartedPayload;
            send this, eValidationStarted, tmpValidationStartedPayload;

            // Request sandbox execution
            send this, eExecuteInSandbox, (sandboxID = tmpTask.id, code = default(seq[int]), inputs = tmpTask.payload);
        }

        // =====================================================================
        // TASK COMPLETION
        // =====================================================================

        on eSandboxExecutionResult do (payload: (
            sandboxID: UUID,
            output: map[string, any],
            resourcesUsed: ResourceUsage
        )) {
            // Find corresponding task
            if (!(payload.sandboxID in runningTasks)) {
                return;
            }

            allValidatorKeys = keys(validators); // Assign to the machine-level variable
            tmpI = 0;
            while (tmpI < sizeof(allValidatorKeys)) {
                tmpValidatorID = allValidatorKeys[tmpI]; // Get the current element
                if (validators[tmpValidatorID].currentTask.value == tmpTask.id.value) {
                    break; // Exit loop if found
                }
                tmpI = tmpI + 1; // Increment loop counter
            }
            tmpResult.taskID = tmpTask.id;
            tmpResult.validator = tmpValidatorID;
            tmpResult.passed = true;
            tmpResult.score = 100;
            tmpResult.detailsPayload = payload.output;
            tmpResult.executionTime = payload.resourcesUsed.cpuMillis;
            tmpResult.resourcesUsed = payload.resourcesUsed;
            tmpResult.timestamp.milliseconds = 0;

            // Move from running to completed
            runningTasks -= tmpTask.id;
            completedTasks[tmpTask.id] = tmpResult;
            totalTasksCompleted = totalTasksCompleted + 1;

            // Update validator state
            if (tmpValidatorID.id != "") {
                tmpValidatorState = validators[tmpValidatorID];
                tmpValidatorState.available = true;
                tmpValidatorState.currentTask.value = "";
                tmpValidatorState.completedCount = tmpValidatorState.completedCount + 1;
                validators[tmpValidatorID] = tmpValidatorState;
            }

            validationResultPayload.result = tmpResult;
            announce eValidationComplete, validationResultPayload;
            send this, eValidationComplete, validationResultPayload;

            // Try to start next task
            TryStartNextTask();
        }

        on eSandboxExecutionFailed do (payload: (sandboxID: UUID, reason: string)) {
            if (!(payload.sandboxID in runningTasks)) {
                return;
            }

            allValidatorKeys = keys(validators); // Assign to the machine-level variable
            tmpI = 0;
            while (tmpI < sizeof(allValidatorKeys)) {
                tmpValidatorID = allValidatorKeys[tmpI]; // Get the current element
                if (validators[tmpValidatorID].currentTask.value == tmpTask.id.value) {
                    break; // Exit loop if found
                }
                tmpI = tmpI + 1; // Increment loop counter
            }

            // Move from running to failed
            runningTasks -= tmpTask.id;
            failedTasks[tmpTask.id] = payload.reason;
            totalTasksFailed = totalTasksFailed + 1;

            // Update validator state
            if (tmpValidatorID.id != "") {
                tmpValidatorState = validators[tmpValidatorID];
                tmpValidatorState.available = true;
                tmpValidatorState.currentTask.value = "";
                tmpValidatorState.failedCount = tmpValidatorState.failedCount + 1;
                validators[tmpValidatorID] = tmpValidatorState;
            }

            announce eValidationFailed, (taskID = tmpTask.id, reason = payload.reason);
            send this, eValidationFailed, (taskID = tmpTask.id, reason = payload.reason);

            // Try to start next task
            TryStartNextTask();
        }

        // =====================================================================
        // TIMEOUT HANDLING
        // =====================================================================

        on eValidationTimeout do (payload: (taskID: UUID)) {
            if (!(payload.taskID in runningTasks)) {
                return;
            }

            tmpTask = runningTasks[payload.taskID];

            // Find validator
            tmpValidatorID.id = "";
            tmpValidatorID.publicKey = default(seq[int]);
            tmpValidatorID.nodeType.typeName = "";
            allValidatorKeys = keys(validators); // Assign to the machine-level variable
            tmpI = 0;
            while (tmpI < sizeof(allValidatorKeys)) {
                tmpValidatorID = allValidatorKeys[tmpI]; // Get the current element
                if (validators[tmpValidatorID].currentTask.value == tmpTask.id.value) {
                    break; // Exit loop if found
                }
                tmpI = tmpI + 1; // Increment loop counter
            }

            // Move from running to failed
            runningTasks -= tmpTask.id;
            failedTasks[tmpTask.id] = "Timeout";
            totalTasksFailed = totalTasksFailed + 1;

            // Update validator state
            if (tmpValidatorID.id != "") {
                tmpValidatorState = validators[tmpValidatorID];
                tmpValidatorState.available = true;
                tmpValidatorState.currentTask.value = "";
                tmpValidatorState.failedCount = tmpValidatorState.failedCount + 1;
                validators[tmpValidatorID] = tmpValidatorState;
            }

            tmpValidationTimeoutPayload.taskID = tmpTask.id;
            announce eValidationTimeout, tmpValidationTimeoutPayload;

            // Try to start next task
            TryStartNextTask();
        }

        // =====================================================================
        // VALIDATOR MANAGEMENT
        // =====================================================================

        on eValidatorAvailable do (payload: (validatorID: NodeID)) {
            if (payload.validatorID in validators) {
                tmpValidatorState = validators[payload.validatorID];
            } else {
                tmpValidatorState.nodeID = payload.validatorID;
                tmpValidatorState.available = true;
                tmpValidatorState.currentTask.value = "";
                tmpValidatorState.completedCount = 0;
                tmpValidatorState.failedCount = 0;
                tmpValidatorState.averageTime = 0;
            }
            tmpValidatorState.available = true;
            validators[payload.validatorID] = tmpValidatorState;

            // Add to validator queue
            validatorQueue += (sizeof(validatorQueue), payload.validatorID);

            // Try to start pending tasks
            TryStartNextTask();
        }

        on eValidatorOverloaded do (payload: (validatorID: NodeID, queueSize: int)) {
            if (payload.validatorID in validators) {
                tmpValidatorState = validators[payload.validatorID];
                tmpValidatorState.available = false;
                validators[payload.validatorID] = tmpValidatorState;
            }
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

    // =====================================================================
    // HELPER FUNCTIONS
    // =====================================================================

    fun InsertTaskByPriority(task: ValidationTask) {
        tmpInserted = false;
        tmpNewQueue = default(seq[ValidationTask]);

        tmpI = 0;
        while (tmpI < sizeof(taskQueue)) {
            if (!tmpInserted && task.priority > taskQueue[tmpI].priority) {
                tmpNewQueue += (sizeof(tmpNewQueue), task);
                tmpInserted = true;
            }
            tmpNewQueue += (sizeof(tmpNewQueue), taskQueue[tmpI]);
            tmpI = tmpI + 1;
        }

        if (!tmpInserted) {
            tmpNewQueue += (sizeof(tmpNewQueue), task);
        }

        taskQueue = tmpNewQueue;
    }

    fun TryStartNextTask() {
        // Check if we can start more tasks
        if (sizeof(runningTasks) >= maxConcurrentTasks) {
            return;
        }

        // Check if there are queued tasks
        if (sizeof(taskQueue) == 0) {
            return;
        }

        // Find available validator
        tmpAvailableValidator.id = "";
        tmpAvailableValidator.publicKey = default(seq[int]);
        tmpAvailableValidator.nodeType.typeName = "";

        allValidatorKeys = keys(validators); // Assign to the machine-level variable
        tmpI = 0;
        while (tmpI < sizeof(allValidatorKeys)) {
            tmpAvailableValidator = allValidatorKeys[tmpI]; // Get the current element
            if (validators[tmpAvailableValidator].available) {
                break; // Exit loop if found
            }
            tmpI = tmpI + 1; // Increment loop counter
        }

        if (tmpAvailableValidator.id == "") {
            return;  // No available validator
        }

        // Start next task
        tmpNextTask = taskQueue[0];

        send this, eStartValidation, (taskID = tmpNextTask.id, validator = tmpAvailableValidator);
    }

    // Statistics helpers
    fun GetQueueSize(): int {
        return sizeof(taskQueue);
    }

    fun GetRunningTaskCount(): int {
        return sizeof(runningTasks);
    }

    fun GetCompletedTaskCount(): int {
        return totalTasksCompleted;
    }

    fun GetFailedTaskCount(): int {
        return totalTasksFailed;
    }

    fun GetSuccessRate(): int {
        tmpTotal = totalTasksCompleted + totalTasksFailed;
        if (tmpTotal == 0) {
            return 100;
        }
        return (totalTasksCompleted * 100) / tmpTotal;
    }
}
