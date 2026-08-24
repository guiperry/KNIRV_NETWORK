/**
 * Test suite for role-based access control
 * This file tests the access control logic implemented in AuthContext
 */

// Mock user data for different roles
const mockUsers = {
  root: { id: 1, username: 'root', email: 'root@knirv.com', role: 'root', permissions: [], created_at: '', updated_at: '' },
  bootnode: { id: 2, username: 'bootnode', email: 'bootnode@knirv.com', role: 'bootnode', permissions: [], created_at: '', updated_at: '' },
  peer: { id: 3, username: 'peer', email: 'peer@knirv.com', role: 'peer', permissions: [], created_at: '', updated_at: '' },
  client: { id: 4, username: 'client', email: 'client@knirv.com', role: 'client', permissions: [], created_at: '', updated_at: '' },
  user: { id: 5, username: 'user', email: 'user@knirv.com', role: 'user', permissions: [], created_at: '', updated_at: '' }
};

// Access control matrices (copied from AuthContext for testing)
const pageAccess: Record<string, string[]> = {
  'root': [
    'dashboard', 'chat', 'monitor', 'models', 'agents', 'skills', 'capabilities', 'properties', 'api', 'settings'
  ],
  'bootnode': [
    'dashboard', 'chat', 'monitor', 'models', 'agents', 'skills', 'capabilities', 'properties', 'api', 'settings'
  ],
  'peer': [
    'dashboard', 'chat', 'monitor', 'models', 'agents', 'skills', 'capabilities', 'properties', 'api', 'settings'
  ],
  'client': [
    'dashboard', 'chat', 'agents', 'skills', 'capabilities', 'settings'
  ],
  'user': [
    'dashboard', 'chat', 'agents', 'skills', 'capabilities', 'settings'
  ]
};

const subPageAccess: Record<string, Record<string, string[]>> = {
  'chat': {
    'root': ['chatchain', 'mychatbrain'],
    'bootnode': ['chatchain', 'mychatbrain'],
    'peer': ['chatchain', 'mychatbrain'],
    'client': ['mychatbrain'],
    'user': ['mychatbrain']
  },
  'monitor': {
    'root': ['network-monitor', 'local-analytics', 'network-explorers'],
    'bootnode': ['network-monitor', 'local-analytics', 'network-explorers'],
    'peer': ['local-analytics', 'network-explorers'],
    'client': ['local-analytics'],
    'user': ['local-analytics']
  },
  'models': {
    'root': ['codex-builder', 'fallback-config', 'dao-voting'],
    'bootnode': ['codex-builder', 'fallback-config', 'dao-voting'],
    'peer': ['codex-builder', 'fallback-config'],
    'client': ['codex-builder'],
    'user': ['codex-builder']
  },
  'agents': {
    'root': ['my-agents', 'my-targets', 'my-workflows'],
    'bootnode': ['my-agents', 'my-targets', 'my-workflows'],
    'peer': ['my-agents', 'my-targets', 'my-workflows'],
    'client': ['my-agents', 'my-workflows'],
    'user': ['my-agents', 'my-workflows']
  },
  'skills': {
    'root': ['skills-dex'],
    'bootnode': ['skills-dex'],
    'peer': ['skills-dex'],
    'client': ['skills-dex'],
    'user': ['skills-dex']
  },
  'capabilities': {
    'root': ['capability-store', 'mcp-manager', 'mcp-servers'],
    'bootnode': ['capability-store', 'mcp-manager', 'mcp-servers'],
    'peer': ['capability-store', 'mcp-manager', 'mcp-servers'],
    'client': ['capability-store', 'mcp-manager'],
    'user': ['capability-store']
  },
  'properties': {
    'root': ['nft-ip-vault'],
    'bootnode': ['nft-ip-vault'],
    'peer': ['nft-ip-vault'],
    'client': ['nft-ip-vault'],
    'user': ['nft-ip-vault']
  },
  'api': {
    'root': ['personal-endpoints'],
    'bootnode': ['personal-endpoints'],
    'peer': ['personal-endpoints'],
    'client': ['personal-endpoints'],
    'user': ['personal-endpoints']
  }
};

// Helper functions (copied from AuthContext logic)
const canAccessPage = (user: any, pageId: string): boolean => {
  if (!user) return false;
  const userRole = user.role?.toLowerCase() || 'user';
  const accessList = pageAccess[userRole] || pageAccess['user'];
  return accessList.includes(pageId);
};

const canAccessSubPage = (user: any, parentPageId: string, subPageId: string): boolean => {
  if (!user) return false;
  const userRole = user.role?.toLowerCase() || 'user';
  const parentAccess = subPageAccess[parentPageId];
  if (!parentAccess) return false;
  const accessList = parentAccess[userRole] || parentAccess['user'] || [];
  return accessList.includes(subPageId);
};

// Test cases
const runAccessControlTests = () => {
  console.log('🧪 Running Access Control Tests...\n');

  // Test 1: Page access for different roles
  console.log('📄 Testing page access...');
  
  // Root should have access to all pages
  const rootUser = mockUsers.root;
  const allPages = ['dashboard', 'chat', 'monitor', 'models', 'agents', 'skills', 'capabilities', 'properties', 'api', 'settings'];
  allPages.forEach(page => {
    const hasAccess = canAccessPage(rootUser, page);
    console.log(`  Root access to ${page}: ${hasAccess ? '✅' : '❌'}`);
  });

  // User should only have limited access
  const regularUser = mockUsers.user;
  const userPages = ['dashboard', 'chat', 'agents', 'skills', 'capabilities', 'settings'];
  const restrictedPages = ['monitor', 'models', 'properties', 'api'];
  
  userPages.forEach(page => {
    const hasAccess = canAccessPage(regularUser, page);
    console.log(`  User access to ${page}: ${hasAccess ? '✅' : '❌'}`);
  });
  
  restrictedPages.forEach(page => {
    const hasAccess = canAccessPage(regularUser, page);
    console.log(`  User access to ${page} (should be denied): ${!hasAccess ? '✅' : '❌'}`);
  });

  console.log('\n📋 Testing sub-page access...');
  
  // Test 2: Sub-page access for different roles
  // Root should access all chat sub-pages
  const chatSubPages = ['chatchain', 'mychatbrain'];
  chatSubPages.forEach(subPage => {
    const hasAccess = canAccessSubPage(rootUser, 'chat', subPage);
    console.log(`  Root access to chat/${subPage}: ${hasAccess ? '✅' : '❌'}`);
  });

  // User should only access mychatbrain
  const hasChainAccess = canAccessSubPage(regularUser, 'chat', 'chatchain');
  const hasBrainAccess = canAccessSubPage(regularUser, 'chat', 'mychatbrain');
  console.log(`  User access to chat/chatchain (should be denied): ${!hasChainAccess ? '✅' : '❌'}`);
  console.log(`  User access to chat/mychatbrain: ${hasBrainAccess ? '✅' : '❌'}`);

  // Test 3: Monitor access restrictions
  const monitorSubPages = ['network-monitor', 'local-analytics', 'network-explorers'];
  monitorSubPages.forEach(subPage => {
    const rootAccess = canAccessSubPage(rootUser, 'monitor', subPage);
    const userAccess = canAccessSubPage(regularUser, 'monitor', subPage);
    console.log(`  Root access to monitor/${subPage}: ${rootAccess ? '✅' : '❌'}`);
    console.log(`  User access to monitor/${subPage} (should be denied): ${!userAccess ? '✅' : '❌'}`);
  });

  // Test 4: Capabilities access levels
  const capabilitySubPages = ['capability-store', 'mcp-manager', 'mcp-servers'];
  const clientUser = mockUsers.client;
  
  capabilitySubPages.forEach(subPage => {
    const rootAccess = canAccessSubPage(rootUser, 'capabilities', subPage);
    const clientAccess = canAccessSubPage(clientUser, 'capabilities', subPage);
    const userAccess = canAccessSubPage(regularUser, 'capabilities', subPage);
    
    console.log(`  Root access to capabilities/${subPage}: ${rootAccess ? '✅' : '❌'}`);
    
    if (subPage === 'capability-store') {
      console.log(`  Client access to capabilities/${subPage}: ${clientAccess ? '✅' : '❌'}`);
      console.log(`  User access to capabilities/${subPage}: ${userAccess ? '✅' : '❌'}`);
    } else {
      console.log(`  Client access to capabilities/${subPage}: ${clientAccess ? '✅' : '❌'}`);
      console.log(`  User access to capabilities/${subPage} (should be denied): ${!userAccess ? '✅' : '❌'}`);
    }
  });

  console.log('\n🎉 Access Control Tests Complete!');
};

// Export for potential use in actual test framework
export { runAccessControlTests, canAccessPage, canAccessSubPage, mockUsers };

// Run tests if this file is executed directly
if (typeof window === 'undefined') {
  runAccessControlTests();
}
