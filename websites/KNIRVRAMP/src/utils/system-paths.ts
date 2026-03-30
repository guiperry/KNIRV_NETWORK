/**
 * Utility functions for determining OS-specific application directories
 */

export interface SystemPaths {
  configDir: string;
  pluginDir: string;
}

/**
 * Get the appropriate config directory for the current OS
 * - Linux: ~/.config/KNIRV-Engine/plugins
 * - Windows: %APPDATA%/KNIRV-Engine/plugins
 * - macOS: ~/Library/Application Support/KNIRV-Engine/plugins
 */
export function getSystemPaths(): SystemPaths {
  const platform = navigator.platform.toLowerCase();
  const userAgent = navigator.userAgent.toLowerCase();

  let configDir: string;

  if (platform.includes('win') || userAgent.includes('windows')) {
    // Windows: %APPDATA%/KNIRV-Engine
    configDir = '%APPDATA%/KNIRV-Engine';
  } else if (platform.includes('mac') || userAgent.includes('mac')) {
    // macOS: ~/Library/Application Support/KNIRV-Engine
    configDir = '~/Library/Application Support/KNIRV-Engine';
  } else {
    // Linux and other Unix-like systems: ~/.config/KNIRV-Engine
    configDir = '~/.config/KNIRV-Engine';
  }

  return {
    configDir,
    pluginDir: `${configDir}/plugins`
  };
}

/**
 * Get a user-friendly description of where the plugin will be saved
 */
export function getPluginLocationDescription(): string {
  const paths = getSystemPaths();
  const platform = navigator.platform.toLowerCase();
  const userAgent = navigator.userAgent.toLowerCase();
  
  if (platform.includes('win') || userAgent.includes('windows')) {
    return `Windows: ${paths.pluginDir.replace('%APPDATA%', 'AppData/Roaming')}`;
  } else if (platform.includes('mac') || userAgent.includes('mac')) {
    return `macOS: ${paths.pluginDir}`;
  } else {
    return `Linux: ${paths.pluginDir}`;
  }
}

/**
 * Get the OS-specific file extension for plugins
 */
export function getPluginExtension(): string {
  const platform = navigator.platform.toLowerCase();
  const userAgent = navigator.userAgent.toLowerCase();
  
  if (platform.includes('win') || userAgent.includes('windows')) {
    return '.dll';
  } else if (platform.includes('mac') || userAgent.includes('mac')) {
    return '.dylib';
  } else {
    return '.so';
  }
}

/**
 * Get the current operating system name
 */
export function getOperatingSystem(): 'windows' | 'mac' | 'linux' {
  const platform = navigator.platform.toLowerCase();
  const userAgent = navigator.userAgent.toLowerCase();
  
  if (platform.includes('win') || userAgent.includes('windows')) {
    return 'windows';
  } else if (platform.includes('mac') || userAgent.includes('mac')) {
    return 'mac';
  } else {
    return 'linux';
  }
}
