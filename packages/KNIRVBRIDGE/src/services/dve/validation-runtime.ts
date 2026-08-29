import { DVE_CONSTANTS, DVETaskType } from '@common/constants/dve.constant';

export type BrowserTaskType =
  | 'policy-check'
  | 'signature-verify'
  | 'reasoning-simple'
  | 'skill-lint';

export interface ValidationTask {
  id: string;
  type: BrowserTaskType | string;
  payload: any;
  parameters?: Record<string, any>;
  timeoutMs?: number;
}

export interface ValidationResult {
  taskID: string;
  status: 'success' | 'failure' | 'error';
  score: number;
  results: any;
  errorMessage?: string;
  executionTimeMs: number;
  signature?: string;
}

/**
 * Runtime validation engine that executes DVE tasks locally in the browser
 * extension service worker context.
 */
export class ValidationRuntime {
  private activeTasks: Map<string, AbortController> = new Map();
  private taskQueue: ValidationTask[] = [];
  private runningCount: number = 0;
  private taskTimestamps: number[] = [];

  /**
   * Execute a single validation task with timeout and concurrency control.
   */
  async executeTask(task: ValidationTask): Promise<ValidationResult> {
    // Rate limit check
    if (!this.checkRateLimit()) {
      return {
        taskID: task.id,
        status: 'error',
        score: 0,
        results: null,
        errorMessage: 'Rate limit exceeded: too many tasks per minute',
        executionTimeMs: 0,
      };
    }

    // Concurrency check
    if (this.runningCount >= DVE_CONSTANTS.MAX_CONCURRENT_TASKS) {
      return new Promise((resolve) => {
        this.taskQueue.push(task);
        // The queued task will be picked up when a slot frees
        setTimeout(() => {
          this.executeTask(task).then(resolve);
        }, 500);
      });
    }

    const timeoutMs = task.timeoutMs ?? DVE_CONSTANTS.TASK_EXECUTION_TIMEOUT_MS;
    const abortController = new AbortController();
    this.activeTasks.set(task.id, abortController);
    this.runningCount++;

    const startTime = performance.now();

    try {
      const result = await this.runWithTimeout(task, timeoutMs, abortController.signal);
      this.recordTaskTimestamp();
      return result;
    } catch (err: any) {
      const elapsedMs = Math.round(performance.now() - startTime);

      if (err.name === 'AbortError' || err.message?.includes('timed out')) {
        return {
          taskID: task.id,
          status: 'error',
          score: 0,
          results: null,
          errorMessage: `Task execution timed out after ${timeoutMs}ms`,
          executionTimeMs: timeoutMs,
        };
      }

      return {
        taskID: task.id,
        status: 'error',
        score: 0,
        results: null,
        errorMessage: err.message || 'Unknown execution error',
        executionTimeMs: elapsedMs,
      };
    } finally {
      this.activeTasks.delete(task.id);
      this.runningCount--;
      this.processQueue();
    }
  }

  /**
   * Cancel a running task.
   */
  cancelTask(taskID: string): boolean {
    const controller = this.activeTasks.get(taskID);
    if (!controller) {
      return false;
    }
    controller.abort();
    this.activeTasks.delete(taskID);
    return true;
  }

  /**
   * Get the number of currently running tasks.
   */
  getRunningCount(): number {
    return this.runningCount;
  }

  /**
   * Get the number of queued tasks.
   */
  getQueueSize(): number {
    return this.taskQueue.length;
  }

  /**
   * Route a task to its type-specific executor.
   */
  private async runWithTimeout(
    task: ValidationTask,
    timeoutMs: number,
    signal: AbortSignal,
  ): Promise<ValidationResult> {
    const startTime = performance.now();

    const executionPromise = this.executeByType(task);
    const timeoutPromise = new Promise<never>((_, reject) => {
      const timer = setTimeout(() => {
        reject(new Error(`Task timed out after ${timeoutMs}ms`));
      }, timeoutMs);

      signal.addEventListener('abort', () => {
        clearTimeout(timer);
        reject(new DOMException('Aborted', 'AbortError'));
      });
    });

    const result = await Promise.race([executionPromise, timeoutPromise]);
    const elapsedMs = Math.round(performance.now() - startTime);

    return {
      ...result,
      taskID: task.id,
      executionTimeMs: elapsedMs,
    };
  }

  /**
   * Dispatch to the appropriate handler based on task type.
   */
  private async executeByType(task: ValidationTask): Promise<ValidationResult> {
    switch (task.type as DVETaskType) {
      case 'policy-check':
        return this.executePolicyCheck(task);
      case 'signature-verify':
        return this.executeSignatureVerify(task);
      case 'reasoning-simple':
        return this.executeReasoningSimple(task);
      case 'skill-lint':
        return this.executeSkillLint(task);
      default:
        return {
          taskID: task.id,
          status: 'error',
          score: 0,
          results: null,
          errorMessage: `Unknown task type: "${task.type}"`,
          executionTimeMs: 0,
        };
    }
  }

  /**
   * Execute a policy-check task.
   * Evaluates whether the given payload conforms to a set of policies.
   */
  private async executePolicyCheck(task: ValidationTask): Promise<ValidationResult> {
    const { payload, parameters } = task;

    if (!payload || !payload.policy || !payload.data) {
      return {
        taskID: task.id,
        status: 'failure',
        score: 0,
        results: null,
        errorMessage: 'Policy-check requires "policy" and "data" fields in payload',
        executionTimeMs: 0,
      };
    }

    const policy = payload.policy;
    const data = payload.data;
    const rules = parameters?.rules || policy.rules || [];

    // Evaluate each rule against the data
    let passedCount = 0;
    const ruleResults: Array<{ rule: string; passed: boolean; detail?: string }> = [];

    for (const rule of rules) {
      const passed = this.evaluatePolicyRule(rule, data);
      if (passed) {
        passedCount++;
      }
      ruleResults.push({ rule: rule.id || rule.name || 'unknown', passed });
    }

    const score = rules.length > 0 ? passedCount / rules.length : 1;
    const allPassed = passedCount === rules.length;

    return {
      taskID: task.id,
      status: allPassed ? 'success' : 'failure',
      score: Math.round(score * 100) / 100,
      results: { ruleResults },
      executionTimeMs: 0, // filled by caller
    };
  }

  /**
   * Execute a signature-verify task.
   * Verifies a cryptographic signature against a public key.
   * In the browser extension context this uses SubtleCrypto.
   */
  private async executeSignatureVerify(task: ValidationTask): Promise<ValidationResult> {
    const { payload } = task;

    if (!payload || !payload.signature || !payload.message || !payload.publicKey) {
      return {
        taskID: task.id,
        status: 'failure',
        score: 0,
        results: null,
        errorMessage: 'Signature-verify requires "signature", "message", and "publicKey" in payload',
        executionTimeMs: 0,
      };
    }

    try {
      const { signature, message, publicKey, algorithm } = payload;
      const algo = algorithm || { name: 'ECDSA', namedCurve: 'P-256', hash: 'SHA-256' };

      // Import the public key
      const keyBuffer = this.hexToArrayBuffer(publicKey);
      const keyData = await crypto.subtle.importKey(
        'raw',
        keyBuffer,
        { name: algo.name, namedCurve: algo.namedCurve || 'P-256' },
        false,
        ['verify'],
      );

      // Decode and verify
      const signatureBuffer = this.hexToArrayBuffer(signature);
      const messageBuffer = new TextEncoder().encode(message);
      const messageHash = await crypto.subtle.digest(algo.hash || 'SHA-256', messageBuffer);
      const isValid = await crypto.subtle.verify(
        { name: algo.name, hash: algo.hash || 'SHA-256' },
        keyData,
        signatureBuffer,
        messageHash,
      );

      return {
        taskID: task.id,
        status: isValid ? 'success' : 'failure',
        score: isValid ? 1 : 0,
        results: { valid: isValid },
        errorMessage: isValid ? undefined : 'Signature verification failed',
        executionTimeMs: 0,
      };
    } catch (err: any) {
      return {
        taskID: task.id,
        status: 'error',
        score: 0,
        results: null,
        errorMessage: `Signature verification error: ${err.message}`,
        executionTimeMs: 0,
      };
    }
  }

  /**
   * Execute a reasoning-simple task.
   * Evaluates a simple logical or mathematical expression against expected output.
   */
  private async executeReasoningSimple(task: ValidationTask): Promise<ValidationResult> {
    const { payload } = task;

    if (!payload || !payload.expression || payload.expected === undefined) {
      return {
        taskID: task.id,
        status: 'failure',
        score: 0,
        results: null,
        errorMessage: 'Reasoning-simple requires "expression" and "expected" fields in payload',
        executionTimeMs: 0,
      };
    }

    try {
      // Safely evaluate mathematical/logical expressions
      const expression = payload.expression;
      let result: any;

      if (typeof expression === 'string') {
        // Use Function constructor for basic math (safer than eval)
        // Only allow arithmetic and boolean operations
        const sanitized = expression.replace(/[^0-9+\-*/().%\s<>!=&|]/g, '');
        result = new Function(`return (${sanitized})`)();
      } else if (typeof expression === 'object' && expression.operation) {
        result = this.evaluateStructuredExpression(expression);
      } else {
        result = expression;
      }

      const expected = payload.expected;
      const isMatch = this.deepEqual(result, expected);

      return {
        taskID: task.id,
        status: isMatch ? 'success' : 'failure',
        score: isMatch ? 1 : 0,
        results: { computed: result, expected },
        errorMessage: isMatch ? undefined : `Expected ${JSON.stringify(expected)}, got ${JSON.stringify(result)}`,
        executionTimeMs: 0,
      };
    } catch (err: any) {
      return {
        taskID: task.id,
        status: 'error',
        score: 0,
        results: null,
        errorMessage: `Reasoning execution error: ${err.message}`,
        executionTimeMs: 0,
      };
    }
  }

  /**
   * Execute a skill-lint task.
   * Validates a skill definition against schema and best practices.
   */
  private async executeSkillLint(task: ValidationTask): Promise<ValidationResult> {
    const { payload } = task;

    if (!payload || !payload.skill) {
      return {
        taskID: task.id,
        status: 'failure',
        score: 0,
        results: null,
        errorMessage: 'Skill-lint requires "skill" field in payload',
        executionTimeMs: 0,
      };
    }

    const skill = payload.skill;
    const issues: string[] = [];
    const warnings: string[] = [];

    // Schema validation
    if (!skill.name || typeof skill.name !== 'string') {
      issues.push('Skill must have a "name" field of type string');
    }

    if (!skill.version || typeof skill.version !== 'string') {
      issues.push('Skill must have a "version" field of type string');
    }

    if (!skill.actions || !Array.isArray(skill.actions)) {
      issues.push('Skill must have an "actions" field of type array');
    }

    // Best practice checks
    if (skill.name && skill.name.length > 64) {
      warnings.push('Skill name exceeds 64 characters (best practice: <= 64)');
    }

    if (skill.description && skill.description.length > 500) {
      warnings.push('Skill description exceeds 500 characters (best practice: <= 500)');
    }

    if (skill.actions && Array.isArray(skill.actions)) {
      for (let i = 0; i < skill.actions.length; i++) {
        const action = skill.actions[i];
        if (!action.name) {
          issues.push(`Action at index ${i} is missing a "name" field`);
        }
        if (!action.handler && !action.url) {
          warnings.push(`Action "${action.name || i}" has neither "handler" nor "url"`);
        }
      }
    }

    const hasIssues = issues.length > 0;
    const hasWarnings = warnings.length > 0;
    const score = hasIssues ? 0 : hasWarnings ? 0.7 : 1;

    return {
      taskID: task.id,
      status: hasIssues ? 'failure' : 'success',
      score,
      results: {
        valid: !hasIssues,
        issues,
        warnings,
        issueCount: issues.length,
        warningCount: warnings.length,
      },
      errorMessage: hasIssues ? `Found ${issues.length} issue(s)` : undefined,
      executionTimeMs: 0,
    };
  }

  /**
   * Process the next task from the queue.
   */
  private processQueue(): void {
    if (this.taskQueue.length > 0 && this.runningCount < DVE_CONSTANTS.MAX_CONCURRENT_TASKS) {
      const nextTask = this.taskQueue.shift()!;
      this.executeTask(nextTask).catch((err) => {
        console.error('Queued task execution failed:', err);
      });
    }
  }

  /**
   * Check the rate limit (max tasks per minute).
   */
  private checkRateLimit(): boolean {
    const now = Date.now();
    const oneMinuteAgo = now - 60_000;

    // Remove timestamps older than 1 minute
    this.taskTimestamps = this.taskTimestamps.filter((t) => t > oneMinuteAgo);

    return this.taskTimestamps.length < DVE_CONSTANTS.MAX_TASKS_PER_MINUTE;
  }

  /**
   * Record a task execution timestamp for rate limiting.
   */
  private recordTaskTimestamp(): void {
    this.taskTimestamps.push(Date.now());
  }

  /**
   * Evaluate a single policy rule against data.
   */
  private evaluatePolicyRule(rule: any, data: any): boolean {
    if (!rule || !rule.condition) {
      return true;
    }

    const { field, operator, value } = rule.condition;
    const fieldValue = this.getNestedValue(data, field);

    switch (operator) {
      case 'eq':
        return fieldValue === value;
      case 'neq':
        return fieldValue !== value;
      case 'gt':
        return Number(fieldValue) > Number(value);
      case 'gte':
        return Number(fieldValue) >= Number(value);
      case 'lt':
        return Number(fieldValue) < Number(value);
      case 'lte':
        return Number(fieldValue) <= Number(value);
      case 'in':
        return Array.isArray(value) && value.includes(fieldValue);
      case 'not_in':
        return Array.isArray(value) && !value.includes(fieldValue);
      case 'contains':
        return typeof fieldValue === 'string' && fieldValue.includes(String(value));
      case 'exists':
        return fieldValue !== undefined && fieldValue !== null;
      default:
        return true;
    }
  }

  /**
   * Get a nested value from an object using dot notation.
   */
  private getNestedValue(obj: any, path: string): any {
    return path.split('.').reduce((current, key) => {
      return current != null ? current[key] : undefined;
    }, obj);
  }

  /**
   * Evaluate a structured expression (operation tree).
   */
  private evaluateStructuredExpression(expr: any): any {
    if (!expr || !expr.operation) {
      return expr;
    }

    const { operation, left, right } = expr;
    const leftVal = this.evaluateStructuredExpression(left);
    const rightVal = this.evaluateStructuredExpression(right);

    switch (operation) {
      case '+': return Number(leftVal) + Number(rightVal);
      case '-': return Number(leftVal) - Number(rightVal);
      case '*': return Number(leftVal) * Number(rightVal);
      case '/': return Number(leftVal) / Number(rightVal);
      case '%': return Number(leftVal) % Number(rightVal);
      case '==': return leftVal === rightVal;
      case '!=': return leftVal !== rightVal;
      case '>': return Number(leftVal) > Number(rightVal);
      case '<': return Number(leftVal) < Number(rightVal);
      case '>=': return Number(leftVal) >= Number(rightVal);
      case '<=': return Number(leftVal) <= Number(rightVal);
      case '&&': return Boolean(leftVal) && Boolean(rightVal);
      case '||': return Boolean(leftVal) || Boolean(rightVal);
      default: return null;
    }
  }

  /**
   * Deep equality comparison for primitive and simple object types.
   */
  private deepEqual(a: any, b: any): boolean {
    if (a === b) {
      return true;
    }

    if (a == null || b == null) {
      return a === b;
    }

    if (typeof a !== typeof b) {
      return false;
    }

    if (typeof a !== 'object') {
      return a === b;
    }

    if (Array.isArray(a) && Array.isArray(b)) {
      if (a.length !== b.length) {
        return false;
      }
      return a.every((val, idx) => this.deepEqual(val, b[idx]));
    }

    const keysA = Object.keys(a);
    const keysB = Object.keys(b);

    if (keysA.length !== keysB.length) {
      return false;
    }

    return keysA.every((key) => this.deepEqual(a[key], b[key]));
  }

  /**
   * Convert a hex string to an ArrayBuffer.
   */
  private hexToArrayBuffer(hex: string): ArrayBuffer {
    const hexStr = hex.replace(/^0x/i, '');
    const bytes = new Uint8Array(hexStr.length / 2);
    for (let i = 0; i < hexStr.length; i += 2) {
      bytes[i / 2] = parseInt(hexStr.substring(i, i + 2), 16);
    }
    return bytes.buffer;
  }
}
