// Mock implementation of targetDiscovery service for testing

const mockTargetDiscoveryService = {
  getCompatibleTargets: jest.fn(() => Promise.resolve([
    {
      id: 'chrome-browser',
      name: 'Chrome Browser',
      type: 'browser',
      category: 'Web Browser',
      status: 'connected',
      description: 'Google Chrome web browser',
      capabilities: ['web_analysis', 'data_extraction', 'content_monitoring'],
      permissions: ['read', 'write'],
      security: 'high',
      icon: () => null, // Mock icon component
      connectionMethod: 'Chrome Extension API',
      platform: 'Desktop'
    },
    {
      id: 'local-filesystem',
      name: 'Local File System',
      type: 'filesystem',
      category: 'Storage',
      status: 'connected',
      description: 'Local file system access',
      capabilities: ['file_analysis', 'document_processing', 'data_extraction'],
      permissions: ['read', 'write'],
      security: 'medium',
      icon: () => null, // Mock icon component
      connectionMethod: 'File System API',
      platform: 'Desktop'
    }
  ])),
  
  discoverTargets: jest.fn(() => Promise.resolve([])),
  refreshTargets: jest.fn(() => Promise.resolve([]))
};

export default mockTargetDiscoveryService;
export { mockTargetDiscoveryService as targetDiscoveryService };
