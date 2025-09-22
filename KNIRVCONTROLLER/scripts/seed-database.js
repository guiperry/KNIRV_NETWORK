#!/usr/bin/env node

/**
 * Database Seeding Script for KNIRVCONTROLLER
 * Creates admin and demo accounts for testing and development
 */

import { authenticationService } from '../src/services/AuthenticationService.js';
import { rxdbService } from '../src/services/RxDBService.js';
import { apiKeyService } from '../src/services/ApiKeyService.js';
import crypto from 'crypto';

const SEED_DATA = {
  admin: {
    username: 'admin',
    email: 'admin@knirv.com',
    password: 'admin123',
    displayName: 'KNIRV Administrator',
    roles: ['admin', 'user'],
    permissions: [
      'admin:all',
      'user:manage',
      'system:configure',
      'profile:read',
      'profile:update',
      'wallet:access',
      'api:manage',
      'deployment:manage'
    ]
  },
  demo: {
    username: 'demo',
    email: 'demo@knirv.com',
    password: 'demo123',
    displayName: 'Demo User',
    roles: ['user'],
    permissions: [
      'profile:read',
      'profile:update',
      'wallet:access'
    ]
  },
  developer: {
    username: 'developer',
    email: 'dev@knirv.com',
    password: 'dev123',
    displayName: 'Developer Account',
    roles: ['developer', 'user'],
    permissions: [
      'profile:read',
      'profile:update',
      'wallet:access',
      'api:create',
      'api:manage',
      'deployment:test'
    ]
  },
  testuser: {
    username: 'testuser',
    email: 'test@example.com',
    password: 'test123',
    displayName: 'Test User',
    roles: ['user'],
    permissions: [
      'profile:read',
      'profile:update',
      'wallet:access'
    ]
  }
};

class DatabaseSeeder {
  constructor() {
    this.isInitialized = false;
  }

  async initialize() {
    if (this.isInitialized) return;

    console.log('🚀 Initializing KNIRVCONTROLLER Database Seeder...');
    
    try {
      // Initialize services
      await rxdbService.initialize();
      await authenticationService.initialize();
      
      this.isInitialized = true;
      console.log('✅ Services initialized successfully');
    } catch (error) {
      console.error('❌ Failed to initialize services:', error);
      throw error;
    }
  }

  async seedUsers() {
    console.log('\n👥 Seeding user accounts...');

    for (const [userType, userData] of Object.entries(SEED_DATA)) {
      try {
        console.log(`\n📝 Creating ${userType} account...`);
        
        // Check if user already exists
        const existingUsers = await rxdbService.findDocuments('users', { email: userData.email });
        if (existingUsers.length > 0) {
          console.log(`⚠️  User ${userData.email} already exists, skipping...`);
          continue;
        }

        // Create user account
        const result = await authenticationService.register({
          username: userData.username,
          email: userData.email,
          password: userData.password,
          displayName: userData.displayName
        });

        if (result.success && result.user) {
          // Update user with additional roles and permissions
          const updatedUser = {
            ...result.user,
            roles: userData.roles,
            permissions: userData.permissions
          };

          await rxdbService.saveDocument('users', updatedUser);

          // Create API key for the user
          const apiKey = await apiKeyService.createApiKey({
            name: `${userData.displayName} API Key`,
            description: `Auto-generated API key for ${userType} account`,
            permissions: userData.permissions,
            expiresAt: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000) // 1 year
          });

          console.log(`✅ Created ${userType} account:`);
          console.log(`   Email: ${userData.email}`);
          console.log(`   Password: ${userData.password}`);
          console.log(`   API Key: ${apiKey.key}`);
          console.log(`   Roles: ${userData.roles.join(', ')}`);
        } else {
          console.error(`❌ Failed to create ${userType} account:`, result.error);
        }
      } catch (error) {
        console.error(`❌ Error creating ${userType} account:`, error);
      }
    }
  }

  async seedSampleData() {
    console.log('\n📊 Seeding sample data...');

    try {
      // Create sample personal graphs
      const sampleGraphs = [
        {
          id: crypto.randomUUID(),
          userId: 'admin-user-id', // Will be updated with actual admin ID
          name: 'Admin Knowledge Graph',
          description: 'Administrator knowledge and capabilities graph',
          nodes: [
            {
              id: 'admin-node-1',
              type: 'capability',
              label: 'System Administration',
              data: { category: 'admin', level: 'expert' }
            },
            {
              id: 'admin-node-2',
              type: 'skill',
              label: 'User Management',
              data: { category: 'admin', level: 'expert' }
            }
          ],
          edges: [
            {
              id: 'admin-edge-1',
              source: 'admin-node-1',
              target: 'admin-node-2',
              type: 'enables'
            }
          ],
          metadata: {
            version: '1.0.0',
            lastSync: new Date().toISOString(),
            capabilities: ['admin', 'management']
          },
          createdAt: new Date(),
          updatedAt: new Date()
        },
        {
          id: crypto.randomUUID(),
          userId: 'demo-user-id', // Will be updated with actual demo ID
          name: 'Demo Knowledge Graph',
          description: 'Demo user capabilities and learning progress',
          nodes: [
            {
              id: 'demo-node-1',
              type: 'capability',
              label: 'Basic Navigation',
              data: { category: 'ui', level: 'beginner' }
            },
            {
              id: 'demo-node-2',
              type: 'skill',
              label: 'Profile Management',
              data: { category: 'user', level: 'intermediate' }
            }
          ],
          edges: [
            {
              id: 'demo-edge-1',
              source: 'demo-node-1',
              target: 'demo-node-2',
              type: 'leads_to'
            }
          ],
          metadata: {
            version: '1.0.0',
            lastSync: new Date().toISOString(),
            capabilities: ['navigation', 'profile']
          },
          createdAt: new Date(),
          updatedAt: new Date()
        }
      ];

      // Get actual user IDs and update graphs
      const adminUsers = await rxdbService.findDocuments('users', { email: 'admin@knirv.com' });
      const demoUsers = await rxdbService.findDocuments('users', { email: 'demo@knirv.com' });

      if (adminUsers.length > 0) {
        sampleGraphs[0].userId = adminUsers[0].id;
        await rxdbService.saveDocument('personal_graphs', sampleGraphs[0]);
        console.log('✅ Created admin knowledge graph');
      }

      if (demoUsers.length > 0) {
        sampleGraphs[1].userId = demoUsers[0].id;
        await rxdbService.saveDocument('personal_graphs', sampleGraphs[1]);
        console.log('✅ Created demo knowledge graph');
      }

    } catch (error) {
      console.error('❌ Error seeding sample data:', error);
    }
  }

  async seedApiKeys() {
    console.log('\n🔑 Seeding additional API keys...');

    try {
      // Create system API keys
      const systemApiKeys = [
        {
          name: 'System Integration Key',
          description: 'API key for system-to-system integration',
          permissions: ['system:read', 'api:access'],
          expiresAt: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000) // 1 year
        },
        {
          name: 'Testing API Key',
          description: 'API key for automated testing',
          permissions: ['test:all', 'api:access'],
          expiresAt: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000) // 90 days
        },
        {
          name: 'Development API Key',
          description: 'API key for development and debugging',
          permissions: ['dev:all', 'api:access', 'debug:access'],
          expiresAt: new Date(Date.now() + 180 * 24 * 60 * 60 * 1000) // 180 days
        }
      ];

      for (const keyData of systemApiKeys) {
        const apiKey = await apiKeyService.createApiKey(keyData);
        console.log(`✅ Created API key: ${keyData.name}`);
        console.log(`   Key: ${apiKey.key}`);
        console.log(`   Permissions: ${keyData.permissions.join(', ')}`);
      }

    } catch (error) {
      console.error('❌ Error seeding API keys:', error);
    }
  }

  async generateReport() {
    console.log('\n📋 Generating seeding report...');

    try {
      const users = await rxdbService.findDocuments('users', {});
      const sessions = await rxdbService.findDocuments('sessions', {});
      const graphs = await rxdbService.findDocuments('personal_graphs', {});
      const apiKeys = await apiKeyService.listApiKeys();

      const report = {
        timestamp: new Date().toISOString(),
        summary: {
          users: users.length,
          sessions: sessions.length,
          personalGraphs: graphs.length,
          apiKeys: apiKeys.length
        },
        accounts: users.map(user => ({
          username: user.username,
          email: user.email,
          roles: user.roles,
          permissions: user.permissions.length
        }))
      };

      console.log('\n📊 Database Seeding Report:');
      console.log('================================');
      console.log(`Timestamp: ${report.timestamp}`);
      console.log(`Users Created: ${report.summary.users}`);
      console.log(`Sessions: ${report.summary.sessions}`);
      console.log(`Personal Graphs: ${report.summary.personalGraphs}`);
      console.log(`API Keys: ${report.summary.apiKeys}`);
      console.log('\n👥 User Accounts:');
      
      report.accounts.forEach(account => {
        console.log(`  • ${account.username} (${account.email})`);
        console.log(`    Roles: ${account.roles.join(', ')}`);
        console.log(`    Permissions: ${account.permissions} total`);
      });

      return report;
    } catch (error) {
      console.error('❌ Error generating report:', error);
      return null;
    }
  }

  async run() {
    try {
      console.log('🌱 Starting KNIRVCONTROLLER Database Seeding...');
      console.log('================================================');

      await this.initialize();
      await this.seedUsers();
      await this.seedSampleData();
      await this.seedApiKeys();
      
      const report = await this.generateReport();

      console.log('\n🎉 Database seeding completed successfully!');
      console.log('\n🔐 Default Credentials:');
      console.log('Admin: admin@knirv.com / admin123');
      console.log('Demo: demo@knirv.com / demo123');
      console.log('Developer: dev@knirv.com / dev123');
      console.log('Test User: test@example.com / test123');
      
      console.log('\n📝 Next Steps:');
      console.log('1. Start the application: npm run dev');
      console.log('2. Login with any of the seeded accounts');
      console.log('3. Test PWA functionality and authentication');
      console.log('4. Deploy to testnet/production environments');

      return report;
    } catch (error) {
      console.error('❌ Database seeding failed:', error);
      process.exit(1);
    }
  }
}

// CLI execution
async function main() {
  const seeder = new DatabaseSeeder();
  
  try {
    await seeder.run();
    process.exit(0);
  } catch (error) {
    console.error('Seeding failed:', error);
    process.exit(1);
  }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export { DatabaseSeeder };
