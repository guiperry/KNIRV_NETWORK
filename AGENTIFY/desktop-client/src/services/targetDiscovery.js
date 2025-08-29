/**
 * Target Discovery Service
 * Discovers available target systems on the local machine
 * This is more appropriate for a desktop application than API calls
 */

import { Globe, Folder, Code, Terminal, Wifi, Image, Smartphone, Database, Monitor } from 'lucide-react';
import {
  isProcessRunning,
  isApplicationInstalled,
  isServiceRunning,
  getMountedDrives,
  getNetworkInterfaces,
  isDatabaseRunning,
  getPlatformInfo
} from './systemUtils';

class TargetDiscoveryService {
  constructor() {
    this.discoveredTargets = [];
    this.lastDiscovery = null;
    this.discoveryInterval = 30000; // 30 seconds
    // Demo mode removed - always use real system detection
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
    
    // Discover media applications
    targets.push(...await this.discoverMediaApplications());
    
    // Discover databases
    targets.push(...await this.discoverDatabases());

    this.discoveredTargets = targets;
    this.lastDiscovery = now;
    
    return targets;
  }

  /**
   * Discover available browsers
   */
  async discoverBrowsers() {
    const browsers = [];
    
    // In a real desktop app, this would check for running browser processes
    // For now, we'll simulate common browsers that might be available
    
    // Check if Chrome is available
    if (await this.isProcessRunning('chrome') || await this.isApplicationInstalled('chrome')) {
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
        platform: this.getPlatform()
      });
    }

    // Check if Firefox is available
    if (await this.isProcessRunning('firefox') || await this.isApplicationInstalled('firefox')) {
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
        connectionMethod: 'WebExtension API',
        platform: this.getPlatform()
      });
    }

    return browsers;
  }

  /**
   * Discover file system access points
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
      capabilities: ['file_analysis', 'document_processing', 'data_extraction'],
      permissions: ['read', 'write'],
      security: 'medium',
      icon: Folder,
      connectionMethod: 'File System API',
      platform: this.getPlatform(),
      path: process.cwd()
    });

    // Check for mounted drives/volumes
    const mountedDrives = await this.getMountedDrives();
    for (const drive of mountedDrives) {
      // Determine drive type for better description
      let driveType = 'External';
      let securityLevel = 'medium';
      
      if (drive.type === 'removable') {
        driveType = 'Removable';
        securityLevel = 'low';
      } else if (drive.type === 'network') {
        driveType = 'Network';
        securityLevel = 'high';
      } else if (drive.type === 'fixed') {
        driveType = 'Fixed';
        securityLevel = 'medium';
      }
      
      fileSystems.push({
        id: `drive-${drive.name}`,
        name: `${drive.name} Drive`,
        type: 'filesystem',
        category: 'Storage',
        status: 'connected',
        description: `${driveType} drive: ${drive.path}`,
        capabilities: ['file_analysis', 'document_processing', 'data_extraction'],
        permissions: ['read', 'write'],
        security: securityLevel,
        icon: Folder,
        connectionMethod: 'File System API',
        platform: this.getPlatform(),
        details: drive
      });
    }

    return fileSystems;
  }

  /**
   * Discover development environments
   */
  async discoverDevelopmentEnvironments() {
    const devEnvs = [];
    
    // Check for VS Code
    if (await this.isProcessRunning('code') || await this.isApplicationInstalled('vscode')) {
      devEnvs.push({
        id: 'vscode-editor',
        name: 'VS Code Editor',
        type: 'application',
        category: 'Development Tool',
        status: 'connected',
        description: 'Visual Studio Code editor',
        capabilities: ['code_analysis', 'quality_assessment', 'bug_detection'],
        permissions: ['read'],
        security: 'high',
        icon: Code,
        connectionMethod: 'VS Code Extension',
        platform: this.getPlatform()
      });
    }

    // Check for terminal/command prompt
    devEnvs.push({
      id: 'system-terminal',
      name: 'System Terminal',
      type: 'system',
      category: 'System Interface',
      status: 'connected',
      description: 'System command line interface',
      capabilities: ['command_execution', 'system_monitoring', 'process_management'],
      permissions: ['read', 'execute'],
      security: 'high',
      icon: Terminal,
      connectionMethod: 'Shell API',
      platform: this.getPlatform()
    });

    return devEnvs;
  }

  /**
   * Discover network interfaces
   */
  async discoverNetworkInterfaces() {
    const networkInterfaces = [];
    
    // Get actual network interfaces
    const interfaces = await this.getNetworkInterfaces();
    
    // Add each interface as a separate target
    for (const iface of interfaces) {
      networkInterfaces.push({
        id: `network-${iface.name}`,
        name: `${iface.name} (${iface.address})`,
        type: 'network',
        category: 'Network Interface',
        status: 'connected',
        description: `Network interface: ${iface.name}, MAC: ${iface.mac}`,
        capabilities: ['network_monitoring', 'security_analysis', 'threat_detection'],
        permissions: ['read'],
        security: 'high',
        icon: Wifi,
        connectionMethod: 'Network API',
        platform: this.getPlatform(),
        details: iface
      });
    }
    
    // If no interfaces were found, add a generic one
    if (networkInterfaces.length === 0) {
      networkInterfaces.push({
        id: 'network-interface',
        name: 'Network Interface',
        type: 'network',
        category: 'Network Monitor',
        status: 'monitoring',
        description: 'Network traffic monitoring',
        capabilities: ['network_monitoring', 'security_analysis', 'threat_detection'],
        permissions: ['read'],
        security: 'high',
        icon: Wifi,
        connectionMethod: 'Network API',
        platform: this.getPlatform()
      });
    }

    return networkInterfaces;
  }

  /**
   * Discover media applications
   */
  async discoverMediaApplications() {
    const mediaApps = [];
    
    // Check for image editing software
    if (await this.isApplicationInstalled('photoshop') || await this.isApplicationInstalled('gimp')) {
      mediaApps.push({
        id: 'image-editor',
        name: 'Image Editor',
        type: 'application',
        category: 'Creative Software',
        status: 'connected',
        description: 'Image editing application',
        capabilities: ['image_processing', 'media_analysis', 'creative_enhancement'],
        permissions: ['read', 'write'],
        security: 'medium',
        icon: Image,
        connectionMethod: 'Plugin API',
        platform: this.getPlatform()
      });
    }

    return mediaApps;
  }

  /**
   * Discover databases
   */
  async discoverDatabases() {
    const databases = [];
    
    // Check for common database types
    const dbTypes = [
      { id: 'postgres', name: 'PostgreSQL' },
      { id: 'mysql', name: 'MySQL' },
      { id: 'mongodb', name: 'MongoDB' },
      { id: 'redis', name: 'Redis' },
      { id: 'sqlite', name: 'SQLite' },
      { id: 'oracle', name: 'Oracle' }
    ];
    
    for (const db of dbTypes) {
      if (await this.isDatabaseRunning(db.id)) {
        databases.push({
          id: `${db.id}-db`,
          name: `${db.name} Database`,
          type: 'database',
          category: 'Database',
          status: 'connected',
          description: `${db.name} database server`,
          capabilities: ['data_mining', 'query_execution', 'data_analysis'],
          permissions: ['read'],
          security: 'high',
          icon: Database,
          connectionMethod: 'Database Driver',
          platform: this.getPlatform(),
          dbType: db.id
        });
      }
    }

    return databases;
  }

  /**
   * Get targets compatible with specific agent capabilities
   */
  async getCompatibleTargets(agentCapabilities = []) {
    const allTargets = await this.discoverTargets();
    
    if (!agentCapabilities || agentCapabilities.length === 0) {
      return allTargets;
    }

    // Normalize agent capabilities for comparison
    const normalizedAgentCaps = agentCapabilities.map(cap => 
      cap.toLowerCase().replace(/\s+/g, '_')
    );

    return allTargets.filter(target => {
      return target.capabilities.some(targetCap => 
        normalizedAgentCaps.some(agentCap => 
          targetCap.includes(agentCap) || 
          agentCap.includes(targetCap) ||
          this.areCapabilitiesRelated(agentCap, targetCap)
        )
      );
    });
  }

  /**
   * Check if two capabilities are related
   */
  areCapabilitiesRelated(cap1, cap2) {
    const relatedCapabilities = {
      'web_analysis': ['data_extraction', 'content_monitoring'],
      'data_extraction': ['web_analysis', 'file_analysis', 'document_processing'],
      'file_analysis': ['document_processing', 'data_extraction'],
      'code_analysis': ['quality_assessment', 'bug_detection'],
      'image_processing': ['media_analysis', 'creative_enhancement']
    };

    return relatedCapabilities[cap1]?.includes(cap2) || 
           relatedCapabilities[cap2]?.includes(cap1);
  }

  // Helper methods for system detection
  async isProcessRunning(processName) {
    return await isProcessRunning(processName);
  }

  async isApplicationInstalled(appName) {
    return await isApplicationInstalled(appName);
  }

  async isServiceRunning(serviceName) {
    return await isServiceRunning(serviceName);
  }

  async getMountedDrives() {
    return await getMountedDrives();
  }

  async getNetworkInterfaces() {
    return await getNetworkInterfaces();
  }

  async isDatabaseRunning(dbType) {
    return await isDatabaseRunning(dbType);
  }

  getPlatform() {
    const platformInfo = getPlatformInfo();
    return `${platformInfo.type} ${platformInfo.release} (${platformInfo.arch})`;
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
export const targetDiscoveryService = new TargetDiscoveryService();
export default targetDiscoveryService;
