/**
 * Enhanced Target Discovery Service
 * Provides robust discovery of available target systems on the local machine
 * with improved process detection, application detection, and service monitoring
 */

import { Globe, Folder, Code, Terminal, Wifi, Image, Smartphone, Database, Monitor } from 'lucide-react';
import { exec } from 'child_process';
import { promisify } from 'util';
import path from 'path';
import os from 'os';

// Promisify exec for async/await usage
const execAsync = promisify(exec);

class EnhancedTargetDiscoveryService {
  constructor() {
    this.discoveredTargets = [];
    this.lastDiscovery = null;
    this.discoveryInterval = 30000; // 30 seconds
    this.demoMode = process.env.AGENTIC_ENGINE_DEMO_MODE === 'true';
    this.platform = os.platform();
  }

  /**
   * Discover all available target systems
   */
  async discoverTargets() {
    const now = Date.now();
    
    // Use cached results if recent
    if (this.lastDiscovery && (now - this.lastDiscovery) < this.discoveryInterval) {
      return this.discoveredTargets;
    }

    const targets = [];

    // Discover browsers
    targets.push(...await this.discoverBrowsers());
    
    // Discover file systems
    targets.push(...await this.discoverFileSystems());
    
    // Discover development environments
    targets.push(...await this.discoverDevelopmentEnvironments());
    
    // Discover network interfaces
    targets.push(...await this.discoverNetworkInterfaces());
    
    // Discover terminals
    targets.push(...await this.discoverTerminals());
    
    // Discover databases
    targets.push(...await this.discoverDatabases());

    this.discoveredTargets = targets;
    this.lastDiscovery = now;
    
    return targets;
  }

  /**
   * Discover available browsers with enhanced detection
   */
  async discoverBrowsers() {
    const browsers = [];
    
    // Check for Chrome
    if (await this.isProcessRunning('chrome') || await this.isApplicationInstalled('chrome')) {
      const version = await this.getApplicationVersion('chrome');
      browsers.push({
        id: 'chrome-browser',
        name: 'Chrome Browser',
        type: 'browser',
        category: 'Web Browser',
        status: 'connected',
        description: 'Google Chrome web browser',
        capabilities: ['web_analysis', 'data_extraction', 'content_monitoring'],
        permissions: ['read', 'write'],
        security: 'high',
        icon: Globe,
        connectionMethod: 'Chrome Extension API',
        platform: this.getPlatform(),
        activeAgents: 0,
        lastActivity: 'Just discovered',
        version: version || 'Unknown',
        dataAccess: ['Browsing History', 'Bookmarks', 'Active Tabs', 'Downloads'],
        restrictions: ['No Private Browsing', 'No Payment Info']
      });
    }

    // Check for Firefox
    if (await this.isProcessRunning('firefox') || await this.isApplicationInstalled('firefox')) {
      const version = await this.getApplicationVersion('firefox');
      browsers.push({
        id: 'firefox-browser',
        name: 'Firefox Browser',
        type: 'browser',
        category: 'Web Browser',
        status: 'connected',
        description: 'Mozilla Firefox web browser',
        capabilities: ['web_analysis', 'data_extraction', 'content_monitoring'],
        permissions: ['read', 'write'],
        security: 'high',
        icon: Globe,
        connectionMethod: 'Firefox Extension API',
        platform: this.getPlatform(),
        activeAgents: 0,
        lastActivity: 'Just discovered',
        version: version || 'Unknown',
        dataAccess: ['Browsing History', 'Bookmarks', 'Active Tabs', 'Downloads'],
        restrictions: ['No Private Browsing', 'No Payment Info']
      });
    }

    // Check for Edge
    if (await this.isProcessRunning('msedge') || await this.isApplicationInstalled('msedge')) {
      const version = await this.getApplicationVersion('msedge');
      browsers.push({
        id: 'edge-browser',
        name: 'Edge Browser',
        type: 'browser',
        category: 'Web Browser',
        status: 'connected',
        description: 'Microsoft Edge web browser',
        capabilities: ['web_analysis', 'data_extraction', 'content_monitoring'],
        permissions: ['read', 'write'],
        security: 'high',
        icon: Globe,
        connectionMethod: 'Edge Extension API',
        platform: this.getPlatform(),
        activeAgents: 0,
        lastActivity: 'Just discovered',
        version: version || 'Unknown',
        dataAccess: ['Browsing History', 'Bookmarks', 'Active Tabs', 'Downloads'],
        restrictions: ['No Private Browsing', 'No Payment Info']
      });
    }

    return browsers;
  }

  /**
   * Discover file systems with enhanced detection
   */
  async discoverFileSystems() {
    const fileSystems = [];
    
    // Local file system is always available
    fileSystems.push({
      id: 'local-filesystem',
      name: 'Local File System',
      type: 'filesystem',
      category: 'Storage',
      status: 'connected',
      description: 'Local file system access',
      capabilities: ['file_analysis', 'directory_scanning', 'content_indexing'],
      permissions: ['read', 'write'],
      security: 'medium',
      icon: Folder,
      connectionMethod: 'File System API',
      platform: this.getPlatform(),
      activeAgents: 0,
      lastActivity: 'Just discovered',
      version: await this.getFilesystemType() || 'Unknown',
      dataAccess: ['Documents', 'Downloads', 'Desktop', 'Pictures'],
      restrictions: ['No System Files', 'No Hidden Folders']
    });

    // Check for mounted drives
    const mountedDrives = await this.getMountedDrives();
    for (const drive of mountedDrives) {
      fileSystems.push({
        id: `filesystem-${drive.name.toLowerCase()}`,
        name: `${drive.name} Drive`,
        type: 'filesystem',
        category: 'External Storage',
        status: 'connected',
        description: `External drive: ${drive.path}`,
        capabilities: ['file_analysis', 'directory_scanning', 'backup'],
        permissions: ['read', 'write'],
        security: 'medium',
        icon: Folder,
        connectionMethod: 'File System API',
        platform: this.getPlatform(),
        activeAgents: 0,
        lastActivity: 'Just discovered',
        version: drive.fsType || 'External',
        dataAccess: ['All Files'],
        restrictions: ['External Drive Limitations']
      });
    }

    return fileSystems;
  }

  /**
   * Discover development environments with enhanced detection
   */
  async discoverDevelopmentEnvironments() {
    const devEnvs = [];
    
    // Check for VS Code
    if (await this.isProcessRunning('code') || await this.isApplicationInstalled('code')) {
      const version = await this.getApplicationVersion('code');
      devEnvs.push({
        id: 'vscode-editor',
        name: 'VS Code Editor',
        type: 'application',
        category: 'Development Tool',
        status: 'connected',
        description: 'Visual Studio Code editor',
        capabilities: ['code_analysis', 'syntax_highlighting', 'error_detection'],
        permissions: ['read', 'write'],
        security: 'high',
        icon: Code,
        connectionMethod: 'VS Code Extension',
        platform: this.getPlatform(),
        activeAgents: 0,
        lastActivity: 'Just discovered',
        version: version || 'Unknown',
        dataAccess: ['Open Files', 'Workspace', 'Git Status'],
        restrictions: ['Workspace Access Only']
      });
    }

    // Check for IntelliJ IDEA
    if (await this.isProcessRunning('idea') || await this.isApplicationInstalled('idea')) {
      const version = await this.getApplicationVersion('idea');
      devEnvs.push({
        id: 'intellij-idea',
        name: 'IntelliJ IDEA',
        type: 'application',
        category: 'Development Tool',
        status: 'connected',
        description: 'JetBrains IntelliJ IDEA IDE',
        capabilities: ['code_analysis', 'syntax_highlighting', 'error_detection', 'refactoring'],
        permissions: ['read', 'write'],
        security: 'high',
        icon: Code,
        connectionMethod: 'IntelliJ Plugin',
        platform: this.getPlatform(),
        activeAgents: 0,
        lastActivity: 'Just discovered',
        version: version || 'Unknown',
        dataAccess: ['Open Files', 'Project Structure', 'Git Status'],
        restrictions: ['Project Access Only']
      });
    }

    return devEnvs;
  }

  /**
   * Discover network interfaces with enhanced detection
   */
  async discoverNetworkInterfaces() {
    const networkInterfaces = [];
    
    // Get network interfaces
    const interfaces = os.networkInterfaces();
    const activeInterfaces = [];
    
    // Process network interfaces
    for (const [name, netInterface] of Object.entries(interfaces)) {
      for (const iface of netInterface) {
        if (!iface.internal) {
          activeInterfaces.push({
            name,
            address: iface.address,
            family: iface.family,
            netmask: iface.netmask
          });
        }
      }
    }
    
    // Add network interface
    networkInterfaces.push({
      id: 'network-interface',
      name: 'Network Interface',
      type: 'network',
      category: 'Network',
      status: 'connected',
      description: 'System network interface',
      capabilities: ['network_monitoring', 'traffic_analysis', 'connectivity_testing'],
      permissions: ['read'],
      security: 'high',
      icon: Wifi,
      connectionMethod: 'Network API',
      platform: this.getPlatform(),
      activeAgents: 0,
      lastActivity: 'Just discovered',
      version: 'System',
      dataAccess: ['Network Status', 'Connection Info'],
      restrictions: ['Read-only Access'],
      interfaces: activeInterfaces
    });

    return networkInterfaces;
  }

  /**
   * Discover terminal/command line interfaces with enhanced detection
   */
  async discoverTerminals() {
    const terminals = [];
    
    // Terminal is always available on most systems
    const isWindows = this.platform === 'win32';
    const terminalName = isWindows ? 'PowerShell' : 'Terminal';
    const terminalProcess = isWindows ? 'powershell' : 'bash';
    
    // Check if the terminal process is running
    const isRunning = await this.isProcessRunning(terminalProcess);
    
    terminals.push({
      id: 'system-terminal',
      name: terminalName,
      type: 'system',
      category: 'System Tool',
      status: isRunning ? 'connected' : 'available',
      description: `System ${terminalName.toLowerCase()}`,
      capabilities: ['command_execution', 'system_monitoring', 'process_management'],
      permissions: ['read', 'write', 'execute'],
      security: 'high',
      icon: Terminal,
      connectionMethod: 'System API',
      platform: this.getPlatform(),
      activeAgents: 0,
      lastActivity: isRunning ? 'Running' : 'Available',
      version: await this.getTerminalVersion(terminalProcess) || 'System',
      dataAccess: ['System Commands', 'Process List'],
      restrictions: ['Admin Rights Required for Some Operations']
    });

    return terminals;
  }

  /**
   * Discover databases with enhanced detection
   */
  async discoverDatabases() {
    const databases = [];
    
    // Check for PostgreSQL
    if (await this.isProcessRunning('postgres') || await this.isServiceRunning('postgresql')) {
      const version = await this.getDatabaseVersion('postgres');
      const connectionStatus = await this.testDatabaseConnection('postgres');
      
      databases.push({
        id: 'postgresql-db',
        name: 'PostgreSQL Database',
        type: 'database',
        category: 'Database',
        status: connectionStatus ? 'connected' : 'available',
        description: 'PostgreSQL database server',
        capabilities: ['data_mining', 'query_execution', 'data_analysis'],
        permissions: ['read'],
        security: 'high',
        icon: Database,
        connectionMethod: 'Database Driver',
        platform: this.getPlatform(),
        activeAgents: 0,
        lastActivity: connectionStatus ? 'Connected' : 'Available',
        version: version || 'Unknown',
        dataAccess: ['Database Tables'],
        restrictions: ['Read-only Access']
      });
    }

    // Check for MySQL/MariaDB
    if (await this.isProcessRunning('mysqld') || await this.isServiceRunning('mysql')) {
      const version = await this.getDatabaseVersion('mysql');
      const connectionStatus = await this.testDatabaseConnection('mysql');
      
      databases.push({
        id: 'mysql-db',
        name: 'MySQL Database',
        type: 'database',
        category: 'Database',
        status: connectionStatus ? 'connected' : 'available',
        description: 'MySQL/MariaDB database server',
        capabilities: ['data_mining', 'query_execution', 'data_analysis'],
        permissions: ['read'],
        security: 'high',
        icon: Database,
        connectionMethod: 'Database Driver',
        platform: this.getPlatform(),
        activeAgents: 0,
        lastActivity: connectionStatus ? 'Connected' : 'Available',
        version: version || 'Unknown',
        dataAccess: ['Database Tables'],
        restrictions: ['Read-only Access']
      });
    }

    return databases;
  }

  /**
   * Enhanced process detection using native Node.js APIs where possible
   * @param {string} processName - The name of the process to check
   * @returns {Promise<boolean>} - True if the process is running
   */
  async isProcessRunning(processName) {
    if (this.demoMode) {
      // For demo purposes, simulate some processes as running
      const runningProcesses = ['chrome', 'code', 'postgres', 'firefox', 'bash', 'powershell'];
      return runningProcesses.includes(processName);
    }
    
    try {
      const platform = this.platform;
      
      if (platform === 'win32') {
        // For Windows, use PowerShell for more reliable results
        const { stdout } = await execAsync('powershell -Command "Get-Process | Select-Object ProcessName"');
        const processes = stdout.split('\\n')
          .map(line => line.trim())
          .filter(line => line && !line.startsWith('---') && line !== 'ProcessName');
        
        return processes.some(proc => 
          proc.toLowerCase() === processName.toLowerCase() || 
          proc.toLowerCase() === `${processName}.exe`.toLowerCase()
        );
      } else {
        // For Unix-like systems, use ps with custom format
        const { stdout } = await execAsync(`ps -A -o comm`);
        const processes = stdout.split('\\n')
          .map(line => line.trim())
          .filter(line => line && line !== 'COMMAND');
        
        return processes.some(proc => {
          const baseName = path.basename(proc);
          return baseName.toLowerCase() === processName.toLowerCase();
        });
      }
    } catch (error) {
      console.warn(`Error checking if process ${processName} is running:`, error);
      
      // Fallback to simpler methods if the above fails
      try {
        const platform = this.platform;
        let command = '';
        
        if (platform === 'win32') {
          command = `tasklist /FI "IMAGENAME eq ${processName}.exe" /NH`;
        } else if (platform === 'darwin') {
          command = `pgrep -x ${processName}`;
        } else {
          // Linux and other Unix-like systems
          command = `pgrep -x ${processName}`;
        }
        
        const { stdout } = await execAsync(command);
        return stdout.trim().length > 0;
      } catch (fallbackError) {
        console.error(`Fallback process check failed for ${processName}:`, fallbackError);
        return false;
      }
    }
  }

  /**
   * Check if an application is installed
   * @param {string} appName - The name of the application to check
   * @returns {Promise<boolean>} - True if the application is installed
   */
  async isApplicationInstalled(appName) {
    if (this.demoMode) {
      // For demo purposes, simulate some applications as installed
      const installedApps = ['chrome', 'firefox', 'code', 'idea', 'msedge'];
      return installedApps.includes(appName.toLowerCase());
    }
    
    try {
      const platform = this.platform;
      
      if (platform === 'win32') {
        // For Windows, check Program Files directories
        const { stdout } = await execAsync('powershell -Command "Get-ItemProperty HKLM:\\\\Software\\\\Microsoft\\\\Windows\\\\CurrentVersion\\\\App*\\\\* | Select-Object DisplayName"');
        const apps = stdout.split('\\n')
          .map(line => line.trim())
          .filter(line => line && !line.startsWith('---') && line !== 'DisplayName');
        
        // Map common app names to potential display names
        const appNameMap = {
          'chrome': ['Google Chrome'],
          'firefox': ['Mozilla Firefox'],
          'code': ['Microsoft Visual Studio Code', 'Visual Studio Code'],
          'msedge': ['Microsoft Edge'],
          'idea': ['IntelliJ IDEA']
        };
        
        const searchNames = appNameMap[appName.toLowerCase()] || [appName];
        
        return apps.some(app => 
          searchNames.some(searchName => 
            app.toLowerCase().includes(searchName.toLowerCase())
          )
        );
      } else if (platform === 'darwin') {
        // For macOS, check Applications directory
        const { stdout } = await execAsync('ls -la /Applications');
        const apps = stdout.split('\\n');
        
        // Map common app names to potential macOS app names
        const appNameMap = {
          'chrome': ['Google Chrome.app'],
          'firefox': ['Firefox.app'],
          'code': ['Visual Studio Code.app'],
          'idea': ['IntelliJ IDEA.app']
        };
        
        const searchNames = appNameMap[appName.toLowerCase()] || [`${appName}.app`];
        
        return apps.some(app => 
          searchNames.some(searchName => 
            app.toLowerCase().includes(searchName.toLowerCase())
          )
        );
      } else {
        // For Linux, check common binary locations
        const { stdout } = await execAsync('which ' + appName);
        return stdout.trim().length > 0;
      }
    } catch (error) {
      console.warn(`Error checking if application ${appName} is installed:`, error);
      return false;
    }
  }

  /**
   * Check if a service is running
   * @param {string} serviceName - The name of the service to check
   * @returns {Promise<boolean>} - True if the service is running
   */
  async isServiceRunning(serviceName) {
    if (this.demoMode) {
      // For demo purposes, simulate some services as running
      const runningServices = ['postgresql', 'mysql', 'nginx', 'apache2'];
      return runningServices.includes(serviceName);
    }
    
    try {
      const platform = this.platform;
      
      if (platform === 'win32') {
        // For Windows, use PowerShell to check services
        const { stdout } = await execAsync(`powershell -Command "Get-Service -Name '*${serviceName}*' | Where-Object {$_.Status -eq 'Running'} | Select-Object Name"`);
        return stdout.trim().length > 0 && !stdout.includes('No matching services found');
      } else if (platform === 'darwin') {
        // For macOS, use launchctl
        const { stdout } = await execAsync(`launchctl list | grep ${serviceName}`);
        return stdout.trim().length > 0;
      } else {
        // For Linux, use systemctl or service command
        try {
          const { stdout: systemctlOut } = await execAsync(`systemctl is-active ${serviceName}`);
          return systemctlOut.trim() === 'active';
        } catch (systemctlError) {
          try {
            const { stdout: serviceOut } = await execAsync(`service ${serviceName} status`);
            return serviceOut.includes('running');
          } catch (serviceError) {
            return false;
          }
        }
      }
    } catch (error) {
      console.warn(`Error checking if service ${serviceName} is running:`, error);
      return false;
    }
  }

  /**
   * Get mounted drives with enhanced detection
   * @returns {Promise<Array>} - Array of mounted drives
   */
  async getMountedDrives() {
    if (this.demoMode) {
      // For demo purposes, return simulated drives
      return [
        { name: 'External', path: '/media/external', fsType: 'exFAT' },
        { name: 'Backup', path: '/media/backup', fsType: 'NTFS' }
      ];
    }
    
    try {
      const platform = this.platform;
      
      if (platform === 'win32') {
        // For Windows, use PowerShell to get drives
        const { stdout } = await execAsync('powershell -Command "Get-PSDrive -PSProvider FileSystem | Select-Object Name, Root"');
        const lines = stdout.split('\\n')
          .map(line => line.trim())
          .filter(line => line && !line.startsWith('---') && !line.startsWith('Name'));
        
        return lines.map(line => {
          const parts = line.split(/\s+/);
          if (parts.length >= 2) {
            return {
              name: parts[0],
              path: parts[1],
              fsType: 'NTFS' // Assume NTFS for Windows
            };
          }
          return null;
        }).filter(Boolean);
      } else if (platform === 'darwin') {
        // For macOS, use diskutil
        const { stdout } = await execAsync('diskutil list');
        const volumes = [];
        
        // Parse diskutil output
        const lines = stdout.split('\\n');
        let currentDisk = null;
        
        for (const line of lines) {
          if (line.startsWith('/dev/')) {
            currentDisk = line.split(' ')[0];
          } else if (line.includes('Apple_HFS') || line.includes('AppleAPFS')) {
            const parts = line.trim().split(/\s+/);
            const name = parts[parts.length - 1];
            if (name !== 'Macintosh HD' && name !== 'System') {
              volumes.push({
                name,
                path: `/Volumes/${name}`,
                fsType: line.includes('Apple_HFS') ? 'HFS+' : 'APFS'
              });
            }
          }
        }
        
        return volumes;
      } else {
        // For Linux, use df
        const { stdout } = await execAsync('df -T');
        const lines = stdout.split('\\n')
          .map(line => line.trim())
          .filter(line => line && !line.startsWith('Filesystem'));
        
        return lines.map(line => {
          const parts = line.split(/\s+/);
          if (parts.length >= 7) {
            const path = parts[6];
            // Only include removable media
            if (path.startsWith('/media/') || path.startsWith('/mnt/')) {
              return {
                name: path.split('/').pop(),
                path,
                fsType: parts[1]
              };
            }
          }
          return null;
        }).filter(Boolean);
      }
    } catch (error) {
      console.warn('Error getting mounted drives:', error);
      return [];
    }
  }

  /**
   * Get the platform information
   * @returns {string} - Platform information
   */
  getPlatform() {
    const platform = this.platform;
    const release = os.release();
    
    if (platform === 'win32') {
      return `Windows ${release}`;
    } else if (platform === 'darwin') {
      return `macOS ${release}`;
    } else if (platform === 'linux') {
      return `Linux ${release}`;
    } else {
      return platform;
    }
  }

  /**
   * Get application version
   * @param {string} appName - The name of the application
   * @returns {Promise<string|null>} - The application version or null if not found
   */
  async getApplicationVersion(appName) {
    if (this.demoMode) {
      // Return simulated versions for demo mode
      const versions = {
        'chrome': '120.0.6099.130',
        'firefox': '121.0.1',
        'code': '1.85.1',
        'idea': '2023.3.2',
        'msedge': '120.0.2210.91'
      };
      return versions[appName.toLowerCase()] || 'Unknown';
    }
    
    try {
      const platform = this.platform;
      
      if (platform === 'win32') {
        // For Windows, use different commands based on the app
        let command = '';
        
        if (appName === 'chrome') {
          command = 'powershell -Command "(Get-Item \'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe\').VersionInfo.FileVersion"';
        } else if (appName === 'firefox') {
          command = 'powershell -Command "(Get-Item \'C:\\Program Files\\Mozilla Firefox\\firefox.exe\').VersionInfo.FileVersion"';
        } else if (appName === 'code') {
          command = 'code --version';
        } else if (appName === 'msedge') {
          command = 'powershell -Command "(Get-Item \'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe\').VersionInfo.FileVersion"';
        } else {
          return null;
        }
        
        const { stdout } = await execAsync(command);
        return stdout.trim().split('\\n')[0];
      } else if (platform === 'darwin') {
        // For macOS, use different commands based on the app
        let command = '';
        
        if (appName === 'chrome') {
          command = '/Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome --version';
        } else if (appName === 'firefox') {
          command = '/Applications/Firefox.app/Contents/MacOS/firefox --version';
        } else if (appName === 'code') {
          command = 'code --version';
        } else {
          return null;
        }
        
        const { stdout } = await execAsync(command);
        // Extract version number
        const versionMatch = stdout.match(/\\d+(\\.\\d+)+/);
        return versionMatch ? versionMatch[0] : stdout.trim();
      } else {
        // For Linux, use different commands based on the app
        let command = '';
        
        if (appName === 'chrome') {
          command = 'google-chrome --version';
        } else if (appName === 'firefox') {
          command = 'firefox --version';
        } else if (appName === 'code') {
          command = 'code --version';
        } else {
          return null;
        }
        
        const { stdout } = await execAsync(command);
        // Extract version number
        const versionMatch = stdout.match(/\\d+(\\.\\d+)+/);
        return versionMatch ? versionMatch[0] : stdout.trim();
      }
    } catch (error) {
      console.warn(`Error getting version for ${appName}:`, error);
      return null;
    }
  }

  /**
   * Get filesystem type
   * @returns {Promise<string|null>} - The filesystem type or null if not found
   */
  async getFilesystemType() {
    if (this.demoMode) {
      const fsTypes = {
        'win32': 'NTFS',
        'darwin': 'APFS',
        'linux': 'ext4'
      };
      return fsTypes[this.platform] || 'Unknown';
    }
    
    try {
      const platform = this.platform;
      
      if (platform === 'win32') {
        // For Windows, assume NTFS for system drive
        return 'NTFS';
      } else if (platform === 'darwin') {
        // For macOS, use diskutil
        const { stdout } = await execAsync('diskutil info / | grep "File System Personality"');
        const match = stdout.match(/File System Personality:\\s+(.+)/);
        return match ? match[1].trim() : null;
      } else {
        // For Linux, use df
        const { stdout } = await execAsync('df -T / | tail -n 1');
        const parts = stdout.trim().split(/\\s+/);
        return parts.length >= 2 ? parts[1] : null;
      }
    } catch (error) {
      console.warn('Error getting filesystem type:', error);
      return null;
    }
  }

  /**
   * Get terminal version
   * @param {string} terminalName - The name of the terminal
   * @returns {Promise<string|null>} - The terminal version or null if not found
   */
  async getTerminalVersion(terminalName) {
    if (this.demoMode) {
      const versions = {
        'powershell': 'PowerShell 7.3.4',
        'bash': 'GNU bash, version 5.1.16'
      };
      return versions[terminalName] || 'Unknown';
    }
    
    try {
      let command = '';
      
      if (terminalName === 'powershell') {
        command = 'powershell -Command "$PSVersionTable.PSVersion"';
      } else if (terminalName === 'bash') {
        command = 'bash --version';
      } else {
        return null;
      }
      
      const { stdout } = await execAsync(command);
      // Extract version information
      if (terminalName === 'powershell') {
        const versionMatch = stdout.match(/Major\\s+:\\s+(\\d+).*Minor\\s+:\\s+(\\d+)/s);
        return versionMatch ? `PowerShell ${versionMatch[1]}.${versionMatch[2]}` : stdout.trim();
      } else {
        const versionMatch = stdout.match(/version\\s+([\\d\\.]+)/i);
        return versionMatch ? `GNU bash, version ${versionMatch[1]}` : stdout.trim();
      }
    } catch (error) {
      console.warn(`Error getting version for ${terminalName}:`, error);
      return null;
    }
  }

  /**
   * Get database version
   * @param {string} dbName - The name of the database
   * @returns {Promise<string|null>} - The database version or null if not found
   */
  async getDatabaseVersion(dbName) {
    if (this.demoMode) {
      const versions = {
        'postgres': 'PostgreSQL 15.4',
        'mysql': 'MySQL 8.0.33'
      };
      return versions[dbName] || 'Unknown';
    }
    
    try {
      let command = '';
      
      if (dbName === 'postgres') {
        command = 'psql --version';
      } else if (dbName === 'mysql') {
        command = 'mysql --version';
      } else {
        return null;
      }
      
      const { stdout } = await execAsync(command);
      // Extract version information
      const versionMatch = stdout.match(/(PostgreSQL|MySQL)\\s+([\\d\\.]+)/i);
      return versionMatch ? `${versionMatch[1]} ${versionMatch[2]}` : stdout.trim();
    } catch (error) {
      console.warn(`Error getting version for ${dbName}:`, error);
      return null;
    }
  }

  /**
   * Test database connection
   * @param {string} dbName - The name of the database
   * @returns {Promise<boolean>} - True if connection is successful
   */
  async testDatabaseConnection(dbName) {
    if (this.demoMode) {
      // For demo purposes, simulate successful connections
      return true;
    }
    
    try {
      let command = '';
      
      if (dbName === 'postgres') {
        // Try to connect to PostgreSQL with a timeout
        command = 'timeout 2 psql -c "SELECT 1" postgres';
      } else if (dbName === 'mysql') {
        // Try to connect to MySQL with a timeout
        command = 'timeout 2 mysql -e "SELECT 1" -u root';
      } else {
        return false;
      }
      
      await execAsync(command);
      return true;
    } catch (error) {
      console.warn(`Error testing connection to ${dbName}:`, error);
      return false;
    }
  }

  /**
   * Force refresh of target discovery
   */
  async refreshTargets() {
    this.lastDiscovery = null;
    return await this.discoverTargets();
  }
}

// Export singleton instance
export const enhancedTargetDiscoveryService = new EnhancedTargetDiscoveryService();
export default enhancedTargetDiscoveryService;