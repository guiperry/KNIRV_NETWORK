/**
 * Tests for TerminalCommandService
 */

import { terminalCommandService } from '../../../src/services/TerminalCommandService';

// Mock fetch globally
global.fetch = jest.fn();

describe('TerminalCommandService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Reset service state
    terminalCommandService['commandHistory'] = [];
    terminalCommandService['currentContext'] = {
      workingDirectory: '/knirv',
      environment: {
        PATH: '/usr/local/bin:/usr/bin:/bin',
        HOME: '/knirv',
        USER: 'knirv'
      },
      user: 'knirv',
      session: 'test-session'
    };
  });

  describe('executeCommand', () => {
    it('should handle empty command', async () => {
      const result = await terminalCommandService.executeCommand('');
      
      expect(result.success).toBe(true);
      expect(result.output).toBe('');
      expect(result.executionTime).toBe(0);
      expect(result.exitCode).toBe(0);
    });

    it('should handle whitespace-only command', async () => {
      const result = await terminalCommandService.executeCommand('   ');
      
      expect(result.success).toBe(true);
      expect(result.output).toBe('');
      expect(result.executionTime).toBe(0);
      expect(result.exitCode).toBe(0);
    });

    it('should add command to history', async () => {
      await terminalCommandService.executeCommand('help');
      
      const history = terminalCommandService.getCommandHistory();
      expect(history).toHaveLength(1);
      expect(history[0].command).toBe('help');
      expect(history[0].result.success).toBe(true);
    });

    it('should limit history to 100 commands', async () => {
      // Add 105 commands
      for (let i = 0; i < 105; i++) {
        await terminalCommandService.executeCommand(`echo ${i}`);
      }
      
      const history = terminalCommandService.getCommandHistory();
      expect(history).toHaveLength(100);
      expect(history[0].command).toBe('echo 5'); // First 5 should be removed
      expect(history[99].command).toBe('echo 104');
    });
  });

  describe('built-in commands', () => {
    describe('help command', () => {
      it('should display help information', async () => {
        const result = await terminalCommandService.executeCommand('help');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('KNIRV Terminal Commands');
        expect(result.output).toContain('System Commands');
        expect(result.output).toContain('KNIRV Commands');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('echo command', () => {
      it('should echo arguments', async () => {
        const result = await terminalCommandService.executeCommand('echo hello world');
        
        expect(result.success).toBe(true);
        expect(result.output).toBe('hello world');
        expect(result.exitCode).toBe(0);
      });

      it('should handle empty echo', async () => {
        const result = await terminalCommandService.executeCommand('echo');
        
        expect(result.success).toBe(true);
        expect(result.output).toBe('');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('pwd command', () => {
      it('should return current working directory', async () => {
        const result = await terminalCommandService.executeCommand('pwd');
        
        expect(result.success).toBe(true);
        expect(result.output).toBe('/knirv');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('cd command', () => {
      it('should change to home directory with no arguments', async () => {
        const result = await terminalCommandService.executeCommand('cd');
        
        expect(result.success).toBe(true);
        expect(result.output).toBe('');
        expect(result.exitCode).toBe(0);
        
        const context = terminalCommandService.getCurrentContext();
        expect(context.workingDirectory).toBe('/knirv');
      });

      it('should change to absolute path', async () => {
        const result = await terminalCommandService.executeCommand('cd /usr/local');
        
        expect(result.success).toBe(true);
        expect(result.exitCode).toBe(0);
        
        const context = terminalCommandService.getCurrentContext();
        expect(context.workingDirectory).toBe('/usr/local');
      });

      it('should change to relative path', async () => {
        const result = await terminalCommandService.executeCommand('cd agents');
        
        expect(result.success).toBe(true);
        expect(result.exitCode).toBe(0);
        
        const context = terminalCommandService.getCurrentContext();
        expect(context.workingDirectory).toBe('/knirv/agents');
      });
    });

    describe('ls command', () => {
      it('should list directory contents', async () => {
        const result = await terminalCommandService.executeCommand('ls');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('agents/');
        expect(result.output).toContain('skills/');
        expect(result.output).toContain('config/');
        expect(result.exitCode).toBe(0);
      });

      it('should list specific directory', async () => {
        const result = await terminalCommandService.executeCommand('ls /usr');
        
        expect(result.success).toBe(true);
        expect(result.exitCode).toBe(0);
      });
    });

    describe('clear command', () => {
      it('should return clear screen sequence', async () => {
        const result = await terminalCommandService.executeCommand('clear');
        
        expect(result.success).toBe(true);
        expect(result.output).toBe('\x1b[2J\x1b[H');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('status command', () => {
      it('should show system status when API is available', async () => {
        (fetch as jest.Mock).mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            cognitive: { running: true },
            agents: { running: true, count: 5 },
            wallet: { connected: true },
            skills: { count: 3 },
            uptime: '12345'
          })
        });

        const result = await terminalCommandService.executeCommand('status');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('KNIRV System Status');
        expect(result.output).toContain('Cognitive Engine: Running');
        expect(result.output).toContain('Active Agents: 5');
        expect(result.exitCode).toBe(0);
      });

      it('should show fallback status when API is unavailable', async () => {
        (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

        const result = await terminalCommandService.executeCommand('status');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Services starting');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('agents command', () => {
      it('should list available agents', async () => {
        const result = await terminalCommandService.executeCommand('agents');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Available agents');
        expect(result.output).toContain('agent_001');
        expect(result.exitCode).toBe(0);
      });

      it('should deploy agent', async () => {
        const result = await terminalCommandService.executeCommand('agents deploy agent_001');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Deploying agent agent_001');
        expect(result.exitCode).toBe(0);
      });

      it('should handle unknown subcommand', async () => {
        const result = await terminalCommandService.executeCommand('agents unknown');
        
        expect(result.success).toBe(false);
        expect(result.error).toContain('Unknown agents subcommand');
        expect(result.exitCode).toBe(1);
      });
    });

    describe('skills command', () => {
      it('should list available skills', async () => {
        const result = await terminalCommandService.executeCommand('skills');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Available skills');
        expect(result.output).toContain('skill_analysis');
        expect(result.exitCode).toBe(0);
      });

      it('should execute skills subcommand', async () => {
        const result = await terminalCommandService.executeCommand('skills invoke skill_analysis');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Skills invoke executed');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('wallet command', () => {
      it('should show wallet status', async () => {
        const result = await terminalCommandService.executeCommand('wallet');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Wallet Status: Connected');
        expect(result.output).toContain('Balance: 1000.00 KNIRV');
        expect(result.output).toContain('NRN Balance: 500.00 NRN');
        expect(result.exitCode).toBe(0);
      });

      it('should execute wallet subcommand', async () => {
        const result = await terminalCommandService.executeCommand('wallet balance');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Wallet balance executed');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('cognitive command', () => {
      it('should show cognitive engine status', async () => {
        const result = await terminalCommandService.executeCommand('cognitive');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Cognitive Engine Status: Running');
        expect(result.output).toContain('Active Skills: 2');
        expect(result.output).toContain('Learning Mode: Active');
        expect(result.exitCode).toBe(0);
      });

      it('should execute cognitive subcommand', async () => {
        const result = await terminalCommandService.executeCommand('cognitive start');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Cognitive engine start executed');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('deploy command', () => {
      it('should deploy with correct arguments', async () => {
        const result = await terminalCommandService.executeCommand('deploy agent test.wasm');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Deploying agent from test.wasm');
        expect(result.exitCode).toBe(0);
      });

      it('should fail with insufficient arguments', async () => {
        const result = await terminalCommandService.executeCommand('deploy agent');
        
        expect(result.success).toBe(false);
        expect(result.error).toBe('Usage: deploy <type> <file>');
        expect(result.exitCode).toBe(1);
      });
    });

    describe('invoke command', () => {
      it('should invoke skill with arguments', async () => {
        const result = await terminalCommandService.executeCommand('invoke analysis_skill param1 param2');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Invoking skill analysis_skill with args: param1 param2');
        expect(result.exitCode).toBe(0);
      });

      it('should fail with no skill specified', async () => {
        const result = await terminalCommandService.executeCommand('invoke');
        
        expect(result.success).toBe(false);
        expect(result.error).toBe('Usage: invoke <skill> [args...]');
        expect(result.exitCode).toBe(1);
      });
    });

    describe('config command', () => {
      it('should show configuration', async () => {
        const result = await terminalCommandService.executeCommand('config');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Configuration:');
        expect(result.output).toContain('cognitive.enabled=true');
        expect(result.exitCode).toBe(0);
      });

      it('should execute config subcommand', async () => {
        const result = await terminalCommandService.executeCommand('config set key value');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Config set executed');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('logs command', () => {
      it('should show logs for all services', async () => {
        const result = await terminalCommandService.executeCommand('logs');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Showing logs for all');
        expect(result.output).toContain('[INFO] System initialized');
        expect(result.exitCode).toBe(0);
      });

      it('should show logs for specific service', async () => {
        const result = await terminalCommandService.executeCommand('logs cognitive');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('Showing logs for cognitive');
        expect(result.exitCode).toBe(0);
      });
    });

    describe('system command', () => {
      it('should show system info', async () => {
        const result = await terminalCommandService.executeCommand('system info');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('KNIRV Controller v1.0.0');
        expect(result.output).toContain('Node.js Runtime');
        expect(result.exitCode).toBe(0);
      });

      it('should execute other system subcommands', async () => {
        const result = await terminalCommandService.executeCommand('system restart');
        
        expect(result.success).toBe(true);
        expect(result.output).toContain('System restart executed');
        expect(result.exitCode).toBe(0);
      });
    });
  });

  describe('remote command execution', () => {
    it('should execute remote command successfully', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          success: true,
          output: 'Remote command output',
          exitCode: 0
        })
      });

      const result = await terminalCommandService.executeCommand('unknown-command arg1 arg2');
      
      expect(result.success).toBe(true);
      expect(result.output).toBe('Remote command output');
      expect(result.exitCode).toBe(0);
      
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:3001/api/terminal/execute',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: expect.stringContaining('unknown-command')
        })
      );
    });

    it('should handle remote command failure', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        statusText: 'Internal Server Error'
      });

      const result = await terminalCommandService.executeCommand('unknown-command');
      
      expect(result.success).toBe(false);
      expect(result.error).toBe("Command 'unknown-command' not found");
      expect(result.exitCode).toBe(127);
    });

    it('should handle network error', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      const result = await terminalCommandService.executeCommand('unknown-command');
      
      expect(result.success).toBe(false);
      expect(result.error).toBe("Command 'unknown-command' not found");
      expect(result.exitCode).toBe(127);
    });
  });

  describe('command parsing', () => {
    it('should parse simple command', async () => {
      const result = await terminalCommandService.executeCommand('echo hello');
      
      expect(result.success).toBe(true);
      expect(result.output).toBe('hello');
    });

    it('should parse command with multiple arguments', async () => {
      const result = await terminalCommandService.executeCommand('echo hello world test');
      
      expect(result.success).toBe(true);
      expect(result.output).toBe('hello world test');
    });

    it('should handle extra whitespace', async () => {
      const result = await terminalCommandService.executeCommand('  echo   hello   world  ');
      
      expect(result.success).toBe(true);
      expect(result.output).toBe('hello world');
    });
  });

  describe('context management', () => {
    it('should return current context', () => {
      const context = terminalCommandService.getCurrentContext();
      
      expect(context.workingDirectory).toBe('/knirv');
      expect(context.user).toBe('knirv');
      expect(context.environment.HOME).toBe('/knirv');
    });

    it('should update working directory', async () => {
      await terminalCommandService.executeCommand('cd /usr/local');
      
      const context = terminalCommandService.getCurrentContext();
      expect(context.workingDirectory).toBe('/usr/local');
    });
  });

  describe('history management', () => {
    it('should clear history', async () => {
      await terminalCommandService.executeCommand('help');
      await terminalCommandService.executeCommand('echo test');
      
      expect(terminalCommandService.getCommandHistory()).toHaveLength(2);
      
      terminalCommandService.clearHistory();
      
      expect(terminalCommandService.getCommandHistory()).toHaveLength(0);
    });

    it('should return copy of history', async () => {
      await terminalCommandService.executeCommand('help');
      
      const history1 = terminalCommandService.getCommandHistory();
      const history2 = terminalCommandService.getCommandHistory();
      
      expect(history1).toEqual(history2);
      expect(history1).not.toBe(history2); // Different objects
    });
  });
});
