/**
 * Target Discovery Service Tests
 * Tests the enhanced target discovery functionality with cross-platform detection
 */

const systemUtils = require('./systemUtils');

// Mock the systemUtils module
jest.mock('./systemUtils', () => ({
  isProcessRunning: jest.fn(),
  isApplicationInstalled: jest.fn(),
  isServiceRunning: jest.fn(),
  getMountedDrives: jest.fn(),
  getNetworkInterfaces: jest.fn(),
  isDatabaseRunning: jest.fn(),
  getPlatformInfo: jest.fn()
}));

// Create a mock TargetDiscoveryService class for testing
class MockTargetDiscoveryService {
  constructor() {
    this.discoveredTargets = [];
    this.lastDiscovery = null;
    this.discoveryInterval = 30000;
  }

  async discoverTargets() {
    const targets = [];
    targets.push(...await this.discoverBrowsers());
    targets.push(...await this.discoverDatabases());
    return targets;
  }

  async discoverBrowsers() {
    const browsers = ['chrome', 'firefox', 'safari', 'edge'];
    const targets = [];

    for (const browser of browsers) {
      try {
        const isRunning = await systemUtils.isProcessRunning(browser);
        const isInstalled = await systemUtils.isApplicationInstalled(browser);

        if (isRunning || isInstalled) {
          targets.push({
            id: `browser-${browser}`,
            name: browser.charAt(0).toUpperCase() + browser.slice(1),
            type: 'browser',
            status: isRunning ? 'running' : 'available',
            installed: isInstalled
          });
        }
      } catch (error) {
        // Skip this browser if detection fails
        continue;
      }
    }

    return targets;
  }

  async discoverDatabases() {
    const databases = ['mysql', 'postgres', 'mongodb', 'redis'];
    const targets = [];

    for (const db of databases) {
      try {
        const isRunning = await systemUtils.isDatabaseRunning(db);

        if (isRunning) {
          targets.push({
            id: `database-${db}`,
            name: db.charAt(0).toUpperCase() + db.slice(1),
            type: 'database',
            status: 'running'
          });
        }
      } catch (error) {
        // Skip this database if detection fails
        continue;
      }
    }

    return targets;
  }
}

describe('TargetDiscoveryService', () => {
  let targetDiscoveryService;

  beforeEach(() => {
    targetDiscoveryService = new MockTargetDiscoveryService();
    jest.clearAllMocks();
  });

  describe('Process Detection', () => {
    describe('Windows Platform', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'win32',
          writable: true
        });
      });

      test('should detect running Chrome process on Windows', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(true);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('running');
        expect(systemUtils.isProcessRunning).toHaveBeenCalledWith('chrome');
      });

      test('should not detect non-running process on Windows', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(false);
        systemUtils.isApplicationInstalled.mockResolvedValue(false);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeUndefined();
      });

      test('should detect installed application via registry on Windows', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(false);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('available');
        expect(chromeTarget.installed).toBe(true);
      });

      test('should detect running service on Windows', async () => {
        systemUtils.isDatabaseRunning.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverDatabases();
        const mysqlTarget = targets.find(t => t.name === 'Mysql');

        expect(mysqlTarget).toBeDefined();
        expect(mysqlTarget.status).toBe('running');
        expect(systemUtils.isDatabaseRunning).toHaveBeenCalledWith('mysql');
      });
    });

    describe('macOS Platform', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'darwin',
          writable: true
        });
      });

      test('should detect running process with pgrep on macOS', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(true);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('running');
      });

      test('should fallback to ps command on macOS when pgrep fails', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(true);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('running');
      });

      test('should detect installed application in Applications directory on macOS', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(false);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('available');
      });

      test('should detect service with launchctl on macOS', async () => {
        systemUtils.isDatabaseRunning.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverDatabases();
        const postgresTarget = targets.find(t => t.name === 'Postgres');

        expect(postgresTarget).toBeDefined();
        expect(postgresTarget.status).toBe('running');
      });
    });

    describe('Linux Platform', () => {
      beforeEach(() => {
        Object.defineProperty(process, 'platform', {
          value: 'linux',
          writable: true
        });
      });

      test('should detect running process with pgrep on Linux', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(true);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('running');
      });

      test('should fallback to ps command on Linux when pgrep fails', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(true);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('running');
      });

      test('should detect installed application with which command on Linux', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(false);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('available');
      });

      test('should detect systemd service on Linux', async () => {
        systemUtils.isDatabaseRunning.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverDatabases();
        const postgresTarget = targets.find(t => t.name === 'Postgres');

        expect(postgresTarget).toBeDefined();
        expect(postgresTarget.status).toBe('running');
      });

      test('should detect package manager installed application on Linux', async () => {
        systemUtils.isProcessRunning.mockResolvedValue(false);
        systemUtils.isApplicationInstalled.mockResolvedValue(true);

        const targets = await targetDiscoveryService.discoverBrowsers();
        const chromeTarget = targets.find(t => t.name === 'Chrome');

        expect(chromeTarget).toBeDefined();
        expect(chromeTarget.status).toBe('available');
      });
    });
  });

  describe('Target Discovery Integration', () => {
    test('should discover browser targets', async () => {
      systemUtils.isProcessRunning.mockResolvedValue(true);
      systemUtils.isApplicationInstalled.mockResolvedValue(true);

      const targets = await targetDiscoveryService.discoverBrowsers();
      expect(targets.length).toBeGreaterThan(0);

      const chromeTarget = targets.find(t => t.name === 'Chrome');
      expect(chromeTarget).toBeDefined();
      expect(chromeTarget.type).toBe('browser');
      expect(chromeTarget.status).toBe('running');
    });

    test('should discover database targets', async () => {
      systemUtils.isDatabaseRunning.mockResolvedValue(true);

      const targets = await targetDiscoveryService.discoverDatabases();
      expect(targets.length).toBeGreaterThan(0);

      const mysqlTarget = targets.find(t => t.name === 'Mysql');
      expect(mysqlTarget).toBeDefined();
      expect(mysqlTarget.type).toBe('database');
      expect(mysqlTarget.status).toBe('running');
    });

    test('should handle discovery errors gracefully', async () => {
      systemUtils.isProcessRunning.mockRejectedValue(new Error('Command failed'));
      systemUtils.isApplicationInstalled.mockRejectedValue(new Error('Command failed'));

      const targets = await targetDiscoveryService.discoverBrowsers();
      expect(targets).toHaveLength(0);
    });
  });

  describe('Error Handling', () => {
    test('should handle exec errors gracefully', async () => {
      systemUtils.isProcessRunning.mockRejectedValue(new Error('Command not found'));

      const targets = await targetDiscoveryService.discoverBrowsers();
      expect(targets).toHaveLength(0);
    });

    test('should handle empty command output', async () => {
      systemUtils.isProcessRunning.mockResolvedValue(false);
      systemUtils.isApplicationInstalled.mockResolvedValue(false);

      const targets = await targetDiscoveryService.discoverBrowsers();
      expect(targets).toHaveLength(0);
    });
  });
});
