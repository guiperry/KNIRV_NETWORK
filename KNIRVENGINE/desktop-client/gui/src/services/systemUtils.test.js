/**
 * System Utils Tests
 * Tests the enhanced cross-platform system detection utilities
 */

const {
  isProcessRunning,
  isApplicationInstalled,
  isServiceRunning,
  getMountedDrives,
  getNetworkInterfaces,
  isDatabaseRunning,
  getPlatformInfo
} = require('./systemUtils');

// Mock child_process.exec for testing
jest.mock('child_process', () => ({
  exec: jest.fn()
}));

const { exec } = require('child_process');
const util = require('util');
const execAsync = util.promisify(exec);

describe('SystemUtils', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('isProcessRunning', () => {
    describe('Windows', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'win32',
          writable: true
        });
      });

      test('should detect running process with exact name match', async () => {
        const mockOutput = `"chrome.exe","1234","Console","1","123,456 K"
"notepad.exe","5678","Console","1","12,345 K"`;

        exec.mockImplementation((command, callback) => {
          callback(null, { stdout: mockOutput });
        });

        const result = await isProcessRunning('chrome');
        expect(result).toBe(true);
      });

      test('should detect process with partial name match', async () => {
        const mockOutput = `"Google Chrome Helper.exe","1234","Console","1","123,456 K"`;

        exec.mockImplementation((command, callback) => {
          callback(null, { stdout: mockOutput });
        });

        const result = await isProcessRunning('chrome');
        expect(result).toBe(true);
      });

      test('should return false for non-existent process', async () => {
        const mockOutput = `"notepad.exe","5678","Console","1","12,345 K"`;

        exec.mockImplementation((command, callback) => {
          callback(null, { stdout: mockOutput });
        });

        const result = await isProcessRunning('nonexistent');
        expect(result).toBe(false);
      });
    });

    describe('macOS', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'darwin',
          writable: true
        });
      });

      test('should detect process with pgrep exact match', async () => {
        exec.mockImplementation((command, callback) => {
          if (command.includes('pgrep -x')) {
            callback(null, { stdout: '1234\n5678\n' });
          }
        });

        const result = await isProcessRunning('chrome');
        expect(result).toBe(true);
      });

      test('should fallback to ps when pgrep fails', async () => {
        exec.mockImplementation((command, callback) => {
          if (command.includes('pgrep -x')) {
            callback(new Error('Not found'), { stdout: '' });
          } else if (command.includes('ps -A')) {
            callback(null, { stdout: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome\n/usr/bin/safari\n' });
          }
        });

        const result = await isProcessRunning('chrome');
        expect(result).toBe(true);
      });

      test('should detect .app bundle processes', async () => {
        exec.mockImplementation((command, callback) => {
          if (command.includes('pgrep -x')) {
            callback(new Error('Not found'), { stdout: '' });
          } else if (command.includes('ps -A')) {
            callback(null, { stdout: '/Applications/Visual Studio Code.app/Contents/MacOS/Electron\n' });
          }
        });

        const result = await isProcessRunning('code');
        expect(result).toBe(true);
      });
    });

    describe('Linux', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'linux',
          writable: true
        });
      });

      test('should detect process with pgrep exact match', async () => {
        exec.mockImplementation((command, callback) => {
          if (command.includes('pgrep -x')) {
            callback(null, { stdout: '1234\n' });
          }
        });

        const result = await isProcessRunning('chrome');
        expect(result).toBe(true);
      });

      test('should fallback to ps when pgrep fails', async () => {
        exec.mockImplementation((command, callback) => {
          if (command.includes('pgrep -x')) {
            callback(new Error('Not found'), { stdout: '' });
          } else if (command.includes('ps -A')) {
            callback(null, { stdout: 'chrome\n/usr/bin/chrome --type=renderer\nfirefox\n' });
          }
        });

        const result = await isProcessRunning('chrome');
        expect(result).toBe(true);
      });

      test('should match process basename', async () => {
        exec.mockImplementation((command, callback) => {
          if (command.includes('pgrep -x')) {
            callback(new Error('Not found'), { stdout: '' });
          } else if (command.includes('ps -A')) {
            callback(null, { stdout: '/usr/bin/google-chrome-stable\n/usr/bin/firefox\n' });
          }
        });

        const result = await isProcessRunning('chrome');
        expect(result).toBe(true);
      });
    });
  });

  describe('isApplicationInstalled', () => {
    describe('Windows', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'win32',
          writable: true
        });
      });

      test('should detect application via registry App Paths', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockRejectedValue(new Error('Not found'));

        exec.mockImplementation((command, callback) => {
          if (command.includes('reg query') && command.includes('App Paths')) {
            callback(null, { stdout: 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\App Paths\\chrome.exe\n    (Default)    REG_SZ    C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe' });
          } else {
            callback(new Error('Not found'), { stdout: '' });
          }
        });

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });

      test('should detect application via PowerShell', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockRejectedValue(new Error('Not found'));

        exec.mockImplementation((command, callback) => {
          if (command.includes('powershell') && command.includes('Get-ItemProperty')) {
            callback(null, { stdout: 'DisplayName\n-----------\nGoogle chrome Browser\n' });
          } else {
            callback(new Error('Not found'), { stdout: '' });
          }
        });

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });
    });

    describe('macOS', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'darwin',
          writable: true
        });
      });

      test('should detect application in Applications directory', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockResolvedValue();

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });

      test('should detect application via mdfind', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockRejectedValue(new Error('Not found'));

        exec.mockImplementation((command, callback) => {
          if (command.includes('mdfind')) {
            callback(null, { stdout: '/Applications/Google Chrome.app\n' });
          } else {
            callback(new Error('Not found'), { stdout: '' });
          }
        });

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });

      test('should detect application via system_profiler', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockRejectedValue(new Error('Not found'));

        exec.mockImplementation((command, callback) => {
          if (command.includes('system_profiler')) {
            callback(null, { stdout: JSON.stringify({
              SPApplicationsDataType: [
                { _name: 'Google Chrome', version: '91.0.4472.114' },
                { _name: 'Safari', version: '14.1.1' }
              ]
            })});
          } else {
            callback(new Error('Not found'), { stdout: '' });
          }
        });

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });

      test('should detect Homebrew installed application', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockRejectedValue(new Error('Not found'));

        exec.mockImplementation((command, callback) => {
          if (command.includes('brew list')) {
            callback(null, { stdout: 'chrome-cli\ngoogle-chrome\nfirefox\n' });
          } else {
            callback(new Error('Not found'), { stdout: '' });
          }
        });

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });
    });

    describe('Linux', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'linux',
          writable: true
        });
      });

      test('should detect application via which command', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockRejectedValue(new Error('Not found'));

        exec.mockImplementation((command, callback) => {
          if (command.includes('which chrome')) {
            callback(null, { stdout: '/usr/bin/chrome\n' });
          } else {
            callback(new Error('Not found'), { stdout: '' });
          }
        });

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });

      test('should detect application via package manager (dpkg)', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockRejectedValue(new Error('Not found'));

        exec.mockImplementation((command, callback) => {
          if (command.includes('dpkg -l')) {
            callback(null, { stdout: 'ii  google-chrome-stable  91.0.4472.114-1  amd64  The web browser from Google\n' });
          } else {
            callback(new Error('Not found'), { stdout: '' });
          }
        });

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });

      test('should detect snap package', async () => {
        const fs = require('fs').promises;
        jest.spyOn(fs, 'access').mockRejectedValue(new Error('Not found'));

        exec.mockImplementation((command, callback) => {
          if (command.includes('snap list')) {
            callback(null, { stdout: 'Name     Version    Rev   Tracking  Publisher   Notes\nchrome   91.0.4472  1234  stable    google✓     -\n' });
          } else {
            callback(new Error('Not found'), { stdout: '' });
          }
        });

        const result = await isApplicationInstalled('chrome');
        expect(result).toBe(true);
      });
    });
  });

  describe('Error Handling', () => {
    test('should handle command execution errors gracefully', async () => {
      exec.mockImplementation((command, callback) => {
        callback(new Error('Command failed'), null);
      });

      const result = await isProcessRunning('test');
      expect(result).toBe(false);
    });

    test('should handle empty output gracefully', async () => {
      exec.mockImplementation((command, callback) => {
        callback(null, { stdout: '' });
      });

      const result = await isProcessRunning('test');
      expect(result).toBe(false);
    });

    test('should handle malformed output gracefully', async () => {
      // Set platform to Windows for consistent CSV parsing test
      Object.defineProperty(process, 'platform', {
        value: 'win32',
        writable: true
      });

      exec.mockImplementation((command, callback) => {
        callback(null, { stdout: 'malformed-output-no-commas-no-quotes' });
      });

      const result = await isProcessRunning('uniqueprocessname');
      expect(result).toBe(false);
    });
  });
});
