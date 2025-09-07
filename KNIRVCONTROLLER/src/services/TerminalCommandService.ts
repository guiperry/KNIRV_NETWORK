/**
 * Terminal Command Processing Service
 * Handles terminal command execution and processing
 */

export interface CommandResult {
  success: boolean;
  output: string;
  error?: string;
  executionTime: number;
  exitCode: number;
}

export interface CommandContext {
  workingDirectory: string;
  environment: Record<string, string>;
  user: string;
  session: string;
}

export interface CommandHistory {
  command: string;
  result: CommandResult;
  timestamp: Date;
  context: CommandContext;
}

export class TerminalCommandService {
  private commandHistory: CommandHistory[] = [];
  private currentContext: CommandContext;
  private baseUrl: string;
  private supportedCommands: Map<string, (args: string[]) => Promise<CommandResult>> = new Map();

  constructor(baseUrl: string = 'http://localhost:3001') {
    this.baseUrl = baseUrl;
    this.currentContext = {
      workingDirectory: '/knirv',
      environment: {
        PATH: '/usr/local/bin:/usr/bin:/bin',
        HOME: '/knirv',
        USER: 'knirv'
      },
      user: 'knirv',
      session: this.generateSessionId()
    };
    this.initializeSupportedCommands();
  }

  private initializeSupportedCommands(): void {
    this.supportedCommands = new Map([
      ['help', this.handleHelpCommand.bind(this)],
      ['ls', this.handleListCommand.bind(this)],
      ['cd', this.handleChangeDirectoryCommand.bind(this)],
      ['pwd', this.handlePrintWorkingDirectoryCommand.bind(this)],
      ['echo', this.handleEchoCommand.bind(this)],
      ['clear', this.handleClearCommand.bind(this)],
      ['status', this.handleStatusCommand.bind(this)],
      ['agents', this.handleAgentsCommand.bind(this)],
      ['skills', this.handleSkillsCommand.bind(this)],
      ['wallet', this.handleWalletCommand.bind(this)],
      ['cognitive', this.handleCognitiveCommand.bind(this)],
      ['deploy', this.handleDeployCommand.bind(this)],
      ['invoke', this.handleInvokeCommand.bind(this)],
      ['config', this.handleConfigCommand.bind(this)],
      ['logs', this.handleLogsCommand.bind(this)],
      ['system', this.handleSystemCommand.bind(this)]
    ]);
  }

  /**
   * Execute a terminal command
   */
  async executeCommand(commandLine: string): Promise<CommandResult> {
    const startTime = Date.now();
    const trimmedCommand = commandLine.trim();

    if (!trimmedCommand) {
      return {
        success: true,
        output: '',
        executionTime: 0,
        exitCode: 0
      };
    }

    try {
      const [command, ...args] = this.parseCommand(trimmedCommand);
      let result: CommandResult;

      // Check if it's a supported built-in command
      if (this.supportedCommands.has(command)) {
        const handler = this.supportedCommands.get(command)!;
        result = await handler(args);
      } else {
        // Try to execute via backend
        result = await this.executeRemoteCommand(command, args);
      }

      const executionTime = Date.now() - startTime;
      result.executionTime = executionTime;

      // Add to history
      const historyEntry: CommandHistory = {
        command: trimmedCommand,
        result,
        timestamp: new Date(),
        context: { ...this.currentContext }
      };
      this.commandHistory.push(historyEntry);

      // Keep only last 100 commands
      if (this.commandHistory.length > 100) {
        this.commandHistory = this.commandHistory.slice(-100);
      }

      return result;
    } catch (error) {
      const executionTime = Date.now() - startTime;
      const errorResult: CommandResult = {
        success: false,
        output: '',
        error: error instanceof Error ? error.message : 'Unknown error',
        executionTime,
        exitCode: 1
      };

      // Add error to history
      const historyEntry: CommandHistory = {
        command: trimmedCommand,
        result: errorResult,
        timestamp: new Date(),
        context: { ...this.currentContext }
      };
      this.commandHistory.push(historyEntry);

      return errorResult;
    }
  }

  private parseCommand(commandLine: string): string[] {
    // Simple command parsing - could be enhanced for complex shell syntax
    return commandLine.split(/\s+/).filter(part => part.length > 0);
  }

  private async executeRemoteCommand(command: string, args: string[]): Promise<CommandResult> {
    try {
      const response = await fetch(`${this.baseUrl}/api/terminal/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          command,
          args,
          context: this.currentContext
        })
      });

      if (!response.ok) {
        throw new Error(`Command execution failed: ${response.statusText}`);
      }

      const result = await response.json();
      return {
        success: result.success,
        output: result.output || '',
        error: result.error,
        executionTime: 0, // Will be set by caller
        exitCode: result.exitCode || 0
      };
    } catch {
      return {
        success: false,
        output: '',
        error: `Command '${command}' not found`,
        executionTime: 0,
        exitCode: 127
      };
    }
  }

  // Built-in command handlers

  private async handleHelpCommand(_args: string[]): Promise<CommandResult> {
    const helpText = `
KNIRV Terminal Commands:

System Commands:
  help                    Show this help message
  status                  Show system status
  clear                   Clear terminal
  pwd                     Print working directory
  cd <dir>                Change directory
  ls [dir]                List directory contents

KNIRV Commands:
  agents                  List available agents
  agents deploy <id>      Deploy an agent
  agents status <id>      Check agent status
  
  skills                  List available skills
  skills invoke <id>      Invoke a skill
  skills status           Show skill execution status
  
  wallet                  Show wallet status
  wallet balance          Show wallet balance
  wallet send <to> <amt>  Send transaction
  
  cognitive               Show cognitive engine status
  cognitive start         Start cognitive engine
  cognitive stop          Stop cognitive engine
  
  deploy <type> <file>    Deploy agent or skill
  invoke <skill> [args]   Invoke skill with arguments
  
  config                  Show configuration
  config set <key> <val>  Set configuration value
  
  logs [service]          Show service logs
  system info             Show system information

Use 'command --help' for detailed help on specific commands.
`;

    return {
      success: true,
      output: helpText.trim(),
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleListCommand(_args: string[]): Promise<CommandResult> {
    // const directory = args[0] || this.currentContext.workingDirectory;
    
    // Mock directory listing - in real implementation would call backend
    const mockListing = [
      'agents/',
      'skills/',
      'config/',
      'logs/',
      'data/',
      'README.md'
    ];

    return {
      success: true,
      output: mockListing.join('\n'),
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleChangeDirectoryCommand(args: string[]): Promise<CommandResult> {
    if (args.length === 0) {
      this.currentContext.workingDirectory = this.currentContext.environment.HOME || '/knirv';
    } else {
      const newDir = args[0];
      if (newDir.startsWith('/')) {
        this.currentContext.workingDirectory = newDir;
      } else {
        this.currentContext.workingDirectory = `${this.currentContext.workingDirectory}/${newDir}`.replace(/\/+/g, '/');
      }
    }

    return {
      success: true,
      output: '',
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handlePrintWorkingDirectoryCommand(_args: string[]): Promise<CommandResult> {
    return {
      success: true,
      output: this.currentContext.workingDirectory,
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleEchoCommand(args: string[]): Promise<CommandResult> {
    return {
      success: true,
      output: args.join(' '),
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleClearCommand(_args: string[]): Promise<CommandResult> {
    return {
      success: true,
      output: '\x1b[2J\x1b[H', // ANSI clear screen
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleStatusCommand(_args: string[]): Promise<CommandResult> {
    try {
      const response = await fetch(`${this.baseUrl}/api/status`);
      if (response.ok) {
        const status = await response.json();
        const output = `
KNIRV System Status:
  Cognitive Engine: ${status.cognitive?.running ? 'Running' : 'Stopped'}
  Agent Manager: ${status.agents?.running ? 'Running' : 'Stopped'}
  Wallet Service: ${status.wallet?.connected ? 'Connected' : 'Disconnected'}
  Active Agents: ${status.agents?.count || 0}
  Active Skills: ${status.skills?.count || 0}
  Uptime: ${status.uptime || 'Unknown'}
`;
        return {
          success: true,
          output: output.trim(),
          executionTime: 0,
          exitCode: 0
        };
      }
    } catch {
      // Fallback status
    }

    return {
      success: true,
      output: 'KNIRV System Status: Services starting...',
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleAgentsCommand(args: string[]): Promise<CommandResult> {
    const subcommand = args[0];
    
    if (!subcommand) {
      return {
        success: true,
        output: 'Available agents:\n  agent_001 - Analysis Agent (Available)\n  agent_002 - Processing Agent (Deployed)',
        executionTime: 0,
        exitCode: 0
      };
    }

    if (subcommand === 'deploy' && args[1]) {
      return {
        success: true,
        output: `Deploying agent ${args[1]}...`,
        executionTime: 0,
        exitCode: 0
      };
    }

    return {
      success: false,
      output: '',
      error: `Unknown agents subcommand: ${subcommand}`,
      executionTime: 0,
      exitCode: 1
    };
  }

  private async handleSkillsCommand(args: string[]): Promise<CommandResult> {
    const subcommand = args[0];
    
    if (!subcommand) {
      return {
        success: true,
        output: 'Available skills:\n  skill_analysis - Data Analysis\n  skill_processing - Text Processing',
        executionTime: 0,
        exitCode: 0
      };
    }

    return {
      success: true,
      output: `Skills ${subcommand} executed`,
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleWalletCommand(args: string[]): Promise<CommandResult> {
    const subcommand = args[0];
    
    if (!subcommand) {
      return {
        success: true,
        output: 'Wallet Status: Connected\nBalance: 1000.00 KNIRV\nNRN Balance: 500.00 NRN',
        executionTime: 0,
        exitCode: 0
      };
    }

    return {
      success: true,
      output: `Wallet ${subcommand} executed`,
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleCognitiveCommand(args: string[]): Promise<CommandResult> {
    const subcommand = args[0];
    
    if (!subcommand) {
      return {
        success: true,
        output: 'Cognitive Engine Status: Running\nActive Skills: 2\nLearning Mode: Active',
        executionTime: 0,
        exitCode: 0
      };
    }

    return {
      success: true,
      output: `Cognitive engine ${subcommand} executed`,
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleDeployCommand(args: string[]): Promise<CommandResult> {
    if (args.length < 2) {
      return {
        success: false,
        output: '',
        error: 'Usage: deploy <type> <file>',
        executionTime: 0,
        exitCode: 1
      };
    }

    return {
      success: true,
      output: `Deploying ${args[0]} from ${args[1]}...`,
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleInvokeCommand(args: string[]): Promise<CommandResult> {
    if (args.length < 1) {
      return {
        success: false,
        output: '',
        error: 'Usage: invoke <skill> [args...]',
        executionTime: 0,
        exitCode: 1
      };
    }

    return {
      success: true,
      output: `Invoking skill ${args[0]} with args: ${args.slice(1).join(' ')}`,
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleConfigCommand(args: string[]): Promise<CommandResult> {
    if (args.length === 0) {
      return {
        success: true,
        output: 'Configuration:\n  cognitive.enabled=true\n  wallet.autoconnect=true\n  terminal.history=100',
        executionTime: 0,
        exitCode: 0
      };
    }

    return {
      success: true,
      output: `Config ${args[0]} executed`,
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleLogsCommand(args: string[]): Promise<CommandResult> {
    const service = args[0] || 'all';
    return {
      success: true,
      output: `Showing logs for ${service}:\n[INFO] System initialized\n[INFO] Services started`,
      executionTime: 0,
      exitCode: 0
    };
  }

  private async handleSystemCommand(args: string[]): Promise<CommandResult> {
    const subcommand = args[0];
    
    if (subcommand === 'info') {
      return {
        success: true,
        output: 'KNIRV Controller v1.0.0\nNode.js Runtime\nMemory: 512MB\nCPU: 2 cores',
        executionTime: 0,
        exitCode: 0
      };
    }

    return {
      success: true,
      output: `System ${subcommand} executed`,
      executionTime: 0,
      exitCode: 0
    };
  }

  /**
   * Get command history
   */
  getCommandHistory(): CommandHistory[] {
    return [...this.commandHistory];
  }

  /**
   * Get current context
   */
  getCurrentContext(): CommandContext {
    return { ...this.currentContext };
  }

  /**
   * Clear command history
   */
  clearHistory(): void {
    this.commandHistory = [];
  }

  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Export singleton instance
export const terminalCommandService = new TerminalCommandService();
