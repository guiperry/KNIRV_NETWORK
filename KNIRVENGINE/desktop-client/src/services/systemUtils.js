/**
 * System Utilities for Target Discovery
 * Provides cross-platform utilities for detecting processes, applications, and system resources
 */

const { exec } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');
const util = require('util');

// Promisify exec for async/await usage
const execAsync = util.promisify(exec);

/**
 * Check if a process is running with enhanced cross-platform detection
 * @param {string} processName - The name of the process to check
 * @returns {Promise<boolean>} - True if the process is running
 */
async function isProcessRunning(processName) {
    try {
        const platform = process.platform;
        const lowerProcessName = processName.toLowerCase();

        if (platform === 'win32') {
            // Enhanced Windows detection using tasklist with CSV format for better parsing
            const { stdout } = await execAsync('tasklist /NH /FO CSV');
            const lines = stdout.trim().split('\n');
            return lines.some(line => {
                const [imageName] = line.split(',');
                const cleanImageName = imageName.replace(/"/g, '').toLowerCase();
                return cleanImageName === `${lowerProcessName}.exe` ||
                       cleanImageName === lowerProcessName ||
                       cleanImageName.includes(lowerProcessName);
            });
        } else if (platform === 'darwin') {
            // Enhanced macOS detection - check both process name and command line
            try {
                // First try exact match with pgrep
                const { stdout: pgrepOut } = await execAsync(`pgrep -x "${processName}"`);
                if (pgrepOut.trim() !== '') return true;
            } catch (e) {
                // pgrep failed, continue with ps
            }

            // Then check with ps for broader matching including .app bundles
            const { stdout } = await execAsync('ps -A -o comm,command');
            const processes = stdout.split('\n').map(line => line.trim());
            return processes.some(proc => {
                const lowerProc = proc.toLowerCase();
                return lowerProc.includes(lowerProcessName) ||
                       lowerProc.includes(`${lowerProcessName}.app`) ||
                       path.basename(proc.split(' ')[0]).toLowerCase() === lowerProcessName;
            });
        } else {
            // Enhanced Linux detection - check both process name and command line
            try {
                // First try exact match with pgrep
                const { stdout: pgrepOut } = await execAsync(`pgrep -x "${processName}"`);
                if (pgrepOut.trim() !== '') return true;
            } catch (e) {
                // pgrep failed, continue with ps
            }

            // Then check with ps for broader matching
            const { stdout } = await execAsync('ps -A -o comm,command');
            const processes = stdout.split('\n').map(line => line.trim());
            return processes.some(proc => {
                const lowerProc = proc.toLowerCase();
                const baseName = path.basename(proc.split(' ')[0]).toLowerCase();
                return baseName === lowerProcessName ||
                       lowerProc.includes(lowerProcessName) ||
                       baseName.startsWith(lowerProcessName);
            });
        }
    } catch (error) {
        console.warn(`Error checking if process ${processName} is running:`, error.message);
        return false;
    }
}

/**
 * Check if an application is installed with enhanced detection
 * @param {string} appName - The name of the application to check
 * @returns {Promise<boolean>} - True if the application is installed
 */
async function isApplicationInstalled(appName) {
    try {
        const platform = process.platform;
        const lowerAppName = appName.toLowerCase();

        if (platform === 'win32') {
            // Enhanced Windows application detection

            // 1. Check Program Files directories with expanded search
            const programFiles = [
                process.env['ProgramFiles'],
                process.env['ProgramFiles(x86)'],
                process.env['LocalAppData'],
                process.env['AppData']
            ].filter(Boolean);

            // Enhanced application path mapping
            const commonPaths = {
                'chrome': ['Google\\Chrome', 'Google Chrome'],
                'firefox': ['Mozilla Firefox', 'Firefox'],
                'vscode': ['Microsoft VS Code', 'VS Code', 'Visual Studio Code'],
                'code': ['Microsoft VS Code', 'VS Code', 'Visual Studio Code'],
                'gimp': ['GIMP 2', 'GIMP'],
                'photoshop': ['Adobe\\Adobe Photoshop', 'Adobe Photoshop'],
                'postgres': ['PostgreSQL'],
                'mysql': ['MySQL'],
                'node': ['nodejs', 'Node.js'],
                'git': ['Git'],
                'docker': ['Docker'],
                'vlc': ['VideoLAN\\VLC'],
                'obs': ['obs-studio']
            };

            const appPaths = commonPaths[lowerAppName] || [appName];

            // Check Program Files directories
            for (const dir of programFiles) {
                for (const appPath of appPaths) {
                    const fullPath = path.join(dir, appPath);
                    if (await pathExists(fullPath)) {
                        return true;
                    }
                }
            }

            // 2. Check Windows Registry for installed applications
            try {
                const { stdout } = await execAsync(`reg query "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\App Paths\\${appName}.exe" /ve`);
                if (stdout.includes('REG_SZ')) return true;
            } catch (e) {
                // Registry key not found, continue
            }

            // 3. Check Uninstall registry entries
            try {
                const { stdout } = await execAsync('reg query "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall" /s /f "' + appName + '"');
                if (stdout.includes(appName)) return true;
            } catch (e) {
                // Registry query failed, continue
            }

            // 4. Use PowerShell to check installed applications
            try {
                const { stdout } = await execAsync(`powershell -Command "Get-ItemProperty HKLM:\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\* | Select-Object DisplayName | Where-Object {$_.DisplayName -like '*${appName}*'}"`);
                if (stdout.includes(appName)) return true;
            } catch (e) {
                // PowerShell failed, continue
            }

            return false;
        } else if (platform === 'darwin') {
            // Enhanced macOS application detection

            // 1. Check Applications directories with expanded search
            const appVariations = [
                appName,
                appName.charAt(0).toUpperCase() + appName.slice(1),
                appName.toUpperCase(),
                // Common app name mappings
                lowerAppName === 'chrome' ? 'Google Chrome' : null,
                lowerAppName === 'vscode' || lowerAppName === 'code' ? 'Visual Studio Code' : null,
                lowerAppName === 'firefox' ? 'Firefox' : null,
                lowerAppName === 'vlc' ? 'VLC' : null
            ].filter(Boolean);

            const searchPaths = [
                '/Applications',
                '/Applications/Utilities',
                `${os.homedir()}/Applications`,
                '/System/Applications',
                '/System/Applications/Utilities'
            ];

            for (const searchPath of searchPaths) {
                for (const variation of appVariations) {
                    const appPath = path.join(searchPath, `${variation}.app`);
                    if (await pathExists(appPath)) {
                        return true;
                    }
                }
            }

            // 2. Use mdfind (Spotlight) for comprehensive search
            try {
                const { stdout } = await execAsync(`mdfind "kMDItemKind == 'Application' && (kMDItemDisplayName == '${appName}' || kMDItemDisplayName == '*${appName}*')"`);
                if (stdout.trim() !== '') return true;
            } catch (e) {
                // mdfind failed, continue
            }

            // 3. Check using system_profiler for installed applications
            try {
                const { stdout } = await execAsync('system_profiler SPApplicationsDataType -json');
                const data = JSON.parse(stdout);
                const applications = data.SPApplicationsDataType || [];
                const found = applications.some(app =>
                    app._name && app._name.toLowerCase().includes(lowerAppName)
                );
                if (found) return true;
            } catch (e) {
                // system_profiler failed, continue
            }

            // 4. Check using which for command-line tools
            try {
                const { stdout } = await execAsync(`which ${appName}`);
                if (stdout.trim() !== '') return true;
            } catch (e) {
                // which failed, continue
            }

            // 5. Check Homebrew installations
            try {
                const { stdout } = await execAsync(`brew list | grep -i ${appName}`);
                if (stdout.trim() !== '') return true;
            } catch (e) {
                // brew not available or app not found
            }

            return false;
        } else {
            // Enhanced Linux application detection

            // 1. Check common binary locations with expanded search
            const binPaths = [
                `/usr/bin/${appName}`,
                `/usr/local/bin/${appName}`,
                `/bin/${appName}`,
                `/opt/${appName}/bin/${appName}`,
                `/opt/${appName}/${appName}`,
                `/snap/bin/${appName}`,
                `${os.homedir()}/.local/bin/${appName}`
            ];

            for (const binPath of binPaths) {
                if (await pathExists(binPath)) {
                    return true;
                }
            }

            // 2. Check using which and whereis
            try {
                const { stdout } = await execAsync(`which ${appName}`);
                if (stdout.trim() !== '') return true;
            } catch (e) {
                // which failed, continue
            }

            try {
                const { stdout } = await execAsync(`whereis ${appName}`);
                const parts = stdout.split(':');
                if (parts.length > 1 && parts[1].trim() !== '') return true;
            } catch (e) {
                // whereis failed, continue
            }

            // 3. Check desktop entries in multiple locations
            const desktopPaths = [
                '/usr/share/applications',
                '/usr/local/share/applications',
                `${os.homedir()}/.local/share/applications`,
                '/var/lib/snapd/desktop/applications'
            ];

            for (const desktopPath of desktopPaths) {
                try {
                    const { stdout } = await execAsync(`find "${desktopPath}" -name "*${appName}*.desktop" -type f 2>/dev/null`);
                    if (stdout.trim() !== '') return true;
                } catch (e) {
                    // find failed for this path, continue
                }
            }

            // 4. Check package managers
            const packageManagers = [
                { cmd: 'dpkg', args: `-l | grep -i ${appName}` },
                { cmd: 'rpm', args: `-qa | grep -i ${appName}` },
                { cmd: 'pacman', args: `-Q | grep -i ${appName}` },
                { cmd: 'snap', args: `list | grep -i ${appName}` },
                { cmd: 'flatpak', args: `list | grep -i ${appName}` }
            ];

            for (const pm of packageManagers) {
                try {
                    const { stdout } = await execAsync(`${pm.cmd} ${pm.args}`);
                    if (stdout.trim() !== '') return true;
                } catch (e) {
                    // Package manager not available or app not found
                }
            }

            // 5. Check AppImage and other portable formats
            try {
                const { stdout } = await execAsync(`find /opt /usr/local -name "*${appName}*" -type f -executable 2>/dev/null`);
                if (stdout.trim() !== '') return true;
            } catch (e) {
                // find failed
            }

            return false;
        }
    } catch (error) {
        console.warn(`Error checking if application ${appName} is installed:`, error);
        return false;
    }
}

/**
 * Check if a service is running with enhanced detection
 * @param {string} serviceName - The name of the service to check
 * @returns {Promise<boolean>} - True if the service is running
 */
async function isServiceRunning(serviceName) {
    try {
        const platform = process.platform;
        const lowerServiceName = serviceName.toLowerCase();

        if (platform === 'win32') {
            // Enhanced Windows service detection

            // 1. Check service status with sc query
            try {
                const { stdout } = await execAsync(`sc query "${serviceName}"`);
                if (stdout.includes('RUNNING')) return true;
            } catch (e) {
                // Service not found with exact name, try variations
            }

            // 2. Try common service name variations
            const serviceVariations = [
                serviceName,
                serviceName.toLowerCase(),
                serviceName.toUpperCase(),
                // Common service mappings
                lowerServiceName === 'postgresql' ? 'postgresql-x64-13' : null,
                lowerServiceName === 'mysql' ? 'MySQL80' : null,
                lowerServiceName === 'apache' ? 'Apache2.4' : null,
                lowerServiceName === 'nginx' ? 'nginx' : null
            ].filter(Boolean);

            for (const variation of serviceVariations) {
                try {
                    const { stdout } = await execAsync(`sc query "${variation}"`);
                    if (stdout.includes('RUNNING')) return true;
                } catch (e) {
                    // Service variation not found
                }
            }

            // 3. Use PowerShell for more comprehensive service detection
            try {
                const { stdout } = await execAsync(`powershell -Command "Get-Service | Where-Object {$_.Name -like '*${serviceName}*' -and $_.Status -eq 'Running'} | Select-Object Name"`);
                if (stdout.includes(serviceName)) return true;
            } catch (e) {
                // PowerShell failed
            }

            return false;
        } else if (platform === 'darwin') {
            // Enhanced macOS service detection

            // 1. Check launchctl services
            try {
                const { stdout } = await execAsync(`launchctl list | grep -i ${serviceName}`);
                if (stdout.trim() !== '') return true;
            } catch (e) {
                // launchctl failed, continue
            }

            // 2. Check for common service variations
            const serviceVariations = [
                serviceName,
                `com.${serviceName}`,
                `org.${serviceName}`,
                // Common macOS service mappings
                lowerServiceName === 'postgresql' ? 'com.edb.launchd.postgresql-13' : null,
                lowerServiceName === 'mysql' ? 'com.oracle.oss.mysql.mysqld' : null,
                lowerServiceName === 'apache' ? 'org.apache.httpd' : null,
                lowerServiceName === 'nginx' ? 'homebrew.mxcl.nginx' : null
            ].filter(Boolean);

            for (const variation of serviceVariations) {
                try {
                    const { stdout } = await execAsync(`launchctl list | grep -i ${variation}`);
                    if (stdout.trim() !== '') return true;
                } catch (e) {
                    // Service variation not found
                }
            }

            // 3. Check Homebrew services
            try {
                const { stdout } = await execAsync(`brew services list | grep -i ${serviceName}`);
                if (stdout.includes('started')) return true;
            } catch (e) {
                // brew services not available
            }

            return false;
        } else {
            // Enhanced Linux service detection

            // 1. Check systemd services
            try {
                const { stdout } = await execAsync(`systemctl is-active ${serviceName}`);
                if (stdout.trim() === 'active') return true;
            } catch (e) {
                // systemctl failed, continue
            }

            // 2. Try service name variations with systemd
            const serviceVariations = [
                serviceName,
                `${serviceName}.service`,
                `${serviceName}d`,
                `${serviceName}d.service`,
                // Common Linux service mappings
                lowerServiceName === 'postgresql' ? 'postgresql.service' : null,
                lowerServiceName === 'mysql' ? 'mysql.service' : null,
                lowerServiceName === 'apache' ? 'apache2.service' : null,
                lowerServiceName === 'nginx' ? 'nginx.service' : null
            ].filter(Boolean);

            for (const variation of serviceVariations) {
                try {
                    const { stdout } = await execAsync(`systemctl is-active ${variation}`);
                    if (stdout.trim() === 'active') return true;
                } catch (e) {
                    // Service variation not found
                }
            }

            // 3. Try SysV init services
            try {
                const { stdout } = await execAsync(`service ${serviceName} status`);
                if (stdout.includes('running') || stdout.includes('active')) return true;
            } catch (e) {
                // SysV service failed, continue
            }

            // 4. Check init.d services
            try {
                const { stdout } = await execAsync(`/etc/init.d/${serviceName} status`);
                if (stdout.includes('running') || stdout.includes('active')) return true;
            } catch (e) {
                // init.d service failed, continue
            }

            // 5. Check Docker services
            try {
                const { stdout } = await execAsync(`docker ps --format "table {{.Names}}" | grep -i ${serviceName}`);
                if (stdout.trim() !== '') return true;
            } catch (e) {
                // Docker not available or service not found
            }

            return false;
        }
    } catch (error) {
        console.warn(`Error checking if service ${serviceName} is running:`, error.message);
        return false;
    }
}

/**
 * Get mounted drives/volumes
 * @returns {Promise<Array>} - Array of mounted drives with name and path
 */
async function getMountedDrives() {
    try {
        const platform = process.platform;
        const drives = [];
        
        if (platform === 'win32') {
            // Windows drives
            const { stdout } = await execAsync('wmic logicaldisk get caption,volumename,drivetype');
            const lines = stdout.split('\n').filter(line => line.trim() !== '');
            
            // Skip header line
            for (let i = 1; i < lines.length; i++) {
                const line = lines[i].trim();
                const parts = line.split(/\s+/);
                
                if (parts.length >= 2) {
                    const driveLetter = parts[0];
                    // DriveType 2=Removable, 3=Fixed, 4=Network, 5=Optical
                    const driveType = parseInt(parts[parts.length - 1], 10);
                    
                    // Only include removable, fixed, and network drives
                    if (driveType >= 2 && driveType <= 4) {
                        let name = parts.slice(1, parts.length - 1).join(' ').trim();
                        if (!name) name = driveLetter;
                        
                        drives.push({
                            name: name,
                            path: driveLetter,
                            type: driveType === 2 ? 'removable' : driveType === 3 ? 'fixed' : 'network'
                        });
                    }
                }
            }
        } else if (platform === 'darwin') {
            // macOS volumes
            const { stdout } = await execAsync('df -h | grep /Volumes/');
            const lines = stdout.split('\n').filter(line => line.trim() !== '');
            
            for (const line of lines) {
                const parts = line.trim().split(/\s+/);
                if (parts.length >= 6) {
                    const path = parts[5];
                    const name = path.replace('/Volumes/', '');
                    
                    drives.push({
                        name: name,
                        path: path,
                        type: 'volume'
                    });
                }
            }
        } else {
            // Linux mounts
            const { stdout } = await execAsync('df -h | grep -E "/media/|/mnt/"');
            const lines = stdout.split('\n').filter(line => line.trim() !== '');
            
            for (const line of lines) {
                const parts = line.trim().split(/\s+/);
                if (parts.length >= 6) {
                    const path = parts[5];
                    const name = path.split('/').pop();
                    
                    drives.push({
                        name: name,
                        path: path,
                        type: 'mount'
                    });
                }
            }
        }
        
        return drives;
    } catch (error) {
        console.warn('Error getting mounted drives:', error);
        return [];
    }
}

/**
 * Get network interfaces
 * @returns {Promise<Array>} - Array of network interfaces
 */
async function getNetworkInterfaces() {
    try {
        const interfaces = os.networkInterfaces();
        const result = [];
        
        for (const [name, netInterface] of Object.entries(interfaces)) {
            for (const iface of netInterface) {
                // Skip internal interfaces
                if (!iface.internal) {
                    result.push({
                        name: name,
                        address: iface.address,
                        family: iface.family,
                        mac: iface.mac
                    });
                }
            }
        }
        
        return result;
    } catch (error) {
        console.warn('Error getting network interfaces:', error);
        return [];
    }
}

/**
 * Check if a database server is running
 * @param {string} dbType - The type of database (mysql, postgres, etc.)
 * @returns {Promise<boolean>} - True if the database server is running
 */
async function isDatabaseRunning(dbType) {
    try {
        // First check if the database process is running
        const processNames = {
            'mysql': ['mysqld', 'mysql'],
            'postgres': ['postgres', 'postgresql'],
            'mongodb': ['mongod'],
            'sqlite': ['sqlite3'],
            'redis': ['redis-server'],
            'oracle': ['oracle', 'tnslsnr']
        };
        
        const processes = processNames[dbType] || [dbType];
        
        for (const proc of processes) {
            if (await isProcessRunning(proc)) {
                return true;
            }
        }
        
        // Then check if the database service is running
        const serviceNames = {
            'mysql': ['mysql', 'mysqld'],
            'postgres': ['postgresql', 'postgres'],
            'mongodb': ['mongodb'],
            'redis': ['redis'],
            'oracle': ['oracle']
        };
        
        const services = serviceNames[dbType] || [dbType];
        
        for (const service of services) {
            if (await isServiceRunning(service)) {
                return true;
            }
        }
        
        // Finally, try to connect to common database ports
        const dbPorts = {
            'mysql': 3306,
            'postgres': 5432,
            'mongodb': 27017,
            'redis': 6379,
            'oracle': 1521
        };
        
        const port = dbPorts[dbType];
        if (port) {
            try {
                const { stdout } = await execAsync(`netstat -an | grep LISTEN | grep :${port}`);
                return stdout.trim() !== '';
            } catch (e) {
                // netstat failed
            }
        }
        
        return false;
    } catch (error) {
        console.warn(`Error checking if database ${dbType} is running:`, error);
        return false;
    }
}

/**
 * Helper function to check if a path exists
 * @param {string} filePath - Path to check
 * @returns {Promise<boolean>} - True if the path exists
 */
async function pathExists(filePath) {
    try {
        await fs.promises.access(filePath);
        return true;
    } catch (error) {
        return false;
    }
}

/**
 * Get the current platform information
 * @returns {Object} - Platform information
 */
function getPlatformInfo() {
    return {
        platform: process.platform,
        arch: process.arch,
        release: os.release(),
        hostname: os.hostname(),
        type: os.type(),
        cpus: os.cpus().length,
        memory: Math.round(os.totalmem() / (1024 * 1024 * 1024)) // GB
    };
}

module.exports = {
    isProcessRunning,
    isApplicationInstalled,
    isServiceRunning,
    getMountedDrives,
    getNetworkInterfaces,
    isDatabaseRunning,
    getPlatformInfo
};