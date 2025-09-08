/**
 * Jest Test Setup
 * Global test configuration and utilities
 */

// Mock console methods to reduce noise during testing
global.console = {
  ...console,
  // Uncomment to suppress console.log during tests
  // log: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
};

// Global test utilities
global.mockExecCommand = (command, output, error = null) => {
  const { exec } = require('child_process');
  exec.mockImplementation((cmd, callback) => {
    if (cmd.includes(command)) {
      if (error) {
        callback(error, null);
      } else {
        callback(null, { stdout: output });
      }
    } else {
      callback(new Error('Command not mocked'), null);
    }
  });
};

// Platform mock utilities
global.mockPlatform = (platform) => {
  Object.defineProperty(process, 'platform', {
    value: platform,
    writable: true
  });
};

// File system mock utilities
global.mockFileExists = (exists = true) => {
  const fs = require('fs').promises;
  if (exists) {
    jest.spyOn(fs, 'access').mockResolvedValue();
  } else {
    jest.spyOn(fs, 'access').mockRejectedValue(new Error('File not found'));
  }
};

// Common test data
global.testData = {
  windows: {
    tasklist: `"chrome.exe","1234","Console","1","123,456 K"
"firefox.exe","5678","Console","1","234,567 K"
"code.exe","9012","Console","1","345,678 K"`,
    registry: 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\App Paths\\chrome.exe\n    (Default)    REG_SZ    C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
    service: 'SERVICE_NAME: postgresql\nSTATE: 4  RUNNING'
  },
  macos: {
    pgrep: '1234\n5678\n',
    ps: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome\n/Applications/Visual Studio Code.app/Contents/MacOS/Electron\n',
    launchctl: '1234\t0\tcom.postgresql.server\n5678\t0\tcom.apple.dock',
    mdfind: '/Applications/Google Chrome.app\n/Applications/Visual Studio Code.app\n',
    systemProfiler: JSON.stringify({
      SPApplicationsDataType: [
        { _name: 'Google Chrome', version: '91.0.4472.114' },
        { _name: 'Visual Studio Code', version: '1.57.1' }
      ]
    })
  },
  linux: {
    pgrep: '1234\n',
    ps: 'chrome\n/usr/bin/chrome --type=renderer\ncode\n/usr/bin/code\n',
    systemctl: 'active\n',
    which: '/usr/bin/chrome\n',
    dpkg: 'ii  google-chrome-stable  91.0.4472.114-1  amd64  The web browser from Google\nii  code  1.57.1-1623937013  amd64  Code editing. Redefined.',
    snap: 'Name     Version    Rev   Tracking  Publisher   Notes\nchrome   91.0.4472  1234  stable    google✓     -\ncode     1.57.1     5678  stable    microsoft✓  classic'
  }
};

// Reset all mocks before each test
beforeEach(() => {
  jest.clearAllMocks();
  jest.resetAllMocks();
});

// Cleanup after each test
afterEach(() => {
  jest.restoreAllMocks();
});
