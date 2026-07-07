#!/usr/bin/env node

/**
 * Database Initialization Script
 * 
 * This script initializes the JSON database with sample data and creates
 * the necessary data structure for the HESK support desk.
 */

const fs = require('fs').promises;
const path = require('path');
const bcrypt = require('bcryptjs');

// Check if already initialized to prevent multiple runs
const INIT_MARKER_FILE = path.join(__dirname, '..', 'support-desk', 'data', '.initialized');

async function checkIfInitialized() {
    try {
        await fs.access(INIT_MARKER_FILE);
        console.log('✅ Support desk database already initialized, skipping...');
        process.exit(0);
    } catch (error) {
        // File doesn't exist, continue with initialization
    }
}

class DatabaseInitializer {
    constructor() {
        this.dataDir = path.join(__dirname, '..', 'support-desk', 'data');
        this.tables = {
            users: [],
            tickets: [],
            replies: [],
            attachments: [],
            categories: [],
            settings: []
        };
    }

    async initialize() {
        console.log('🗄️  Initializing JSON Database...');
        console.log('================================');

        try {
            // Create data directory
            await this.createDataDirectory();
            
            // Initialize tables with sample data
            await this.initializeUsers();
            await this.initializeCategories();
            await this.initializeSettings();
            await this.initializeSampleTickets();
            
            // Write all tables to JSON files
            await this.writeAllTables();
            
            console.log('\n✅ Database initialization completed successfully!');
            this.printSummary();
            
        } catch (error) {
            console.error('\n❌ Database initialization failed:', error.message);
            console.error(error.stack);
            process.exit(1);
        }
    }

    async createDataDirectory() {
        try {
            await fs.access(this.dataDir);
            console.log('📁 Data directory already exists');
        } catch {
            await fs.mkdir(this.dataDir, { recursive: true });
            console.log('📁 Created data directory');
        }
    }

    async initializeUsers() {
        console.log('\n👥 Initializing users...');
        
        // Create admin user
        const adminPassword = await bcrypt.hash('admin123', 10);
        const adminUser = {
            id: 1,
            username: 'admin',
            email: 'admin@knirv.com',
            password: adminPassword,
            name: 'KNIRV Administrator',
            is_admin: true,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString()
        };

        // Create sample customer user
        const customerPassword = await bcrypt.hash('customer123', 10);
        const customerUser = {
            id: 2,
            username: 'customer',
            email: 'customer@example.com',
            password: customerPassword,
            name: 'Sample Customer',
            is_admin: false,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString()
        };

        this.tables.users = [adminUser, customerUser];
        console.log('✅ Created admin and sample customer users');
        console.log('   Admin: admin@knirv.com / admin123');
        console.log('   Customer: customer@example.com / customer123');
    }

    async initializeCategories() {
        console.log('\n📂 Initializing categories...');
        
        const categories = [
            {
                id: 1,
                name: 'General Support',
                description: 'General questions and support requests',
                color: '#3498db',
                order: 1,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 2,
                name: 'Technical Issue',
                description: 'Technical problems and bug reports',
                color: '#e74c3c',
                order: 2,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 3,
                name: 'Billing',
                description: 'Billing and payment related questions',
                color: '#f39c12',
                order: 3,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 4,
                name: 'Feature Request',
                description: 'Suggestions for new features',
                color: '#2ecc71',
                order: 4,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            }
        ];

        this.tables.categories = categories;
        console.log(`✅ Created ${categories.length} categories`);
    }

    async initializeSettings() {
        console.log('\n⚙️  Initializing settings...');
        
        const settings = [
            {
                id: 1,
                key: 'site_title',
                value: 'KNIRV Support Desk',
                description: 'Main site title',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 2,
                key: 'admin_email',
                value: 'admin@knirv.com',
                description: 'Administrator email address',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 3,
                key: 'max_file_size',
                value: '10485760',
                description: 'Maximum file upload size in bytes',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 4,
                key: 'allowed_file_types',
                value: 'image/jpeg,image/png,image/gif,text/plain,application/pdf',
                description: 'Allowed file types for uploads',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 5,
                key: 'auto_close_days',
                value: '30',
                description: 'Days after which resolved tickets are auto-closed',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            }
        ];

        this.tables.settings = settings;
        console.log(`✅ Created ${settings.length} settings`);
    }

    async initializeSampleTickets() {
        console.log('\n🎫 Initializing sample tickets...');
        
        const sampleTickets = [
            {
                id: 1,
                trackid: 'KNIRV-001',
                name: 'Sample Customer',
                email: 'customer@example.com',
                category: 'general',
                priority: 'medium',
                subject: 'Welcome to KNIRV Support',
                message: 'This is a sample ticket to demonstrate the support system functionality.',
                status: 'new',
                created_at: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(), // 1 day ago
                updated_at: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString()
            },
            {
                id: 2,
                trackid: 'KNIRV-002',
                name: 'John Doe',
                email: 'john@example.com',
                category: 'technical',
                priority: 'high',
                subject: 'Login Issues',
                message: 'I am unable to log into my account. The password reset is not working.',
                status: 'replied',
                created_at: new Date(Date.now() - 12 * 60 * 60 * 1000).toISOString(), // 12 hours ago
                updated_at: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString() // 6 hours ago
            }
        ];

        // Sample replies
        const sampleReplies = [
            {
                id: 1,
                ticket_id: 2,
                name: 'KNIRV Support',
                email: 'admin@knirv.com',
                message: 'Thank you for contacting us. We are looking into your login issue and will have it resolved shortly.',
                is_admin: true,
                created_at: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
                updated_at: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString()
            }
        ];

        this.tables.tickets = sampleTickets;
        this.tables.replies = sampleReplies;
        
        console.log(`✅ Created ${sampleTickets.length} sample tickets`);
        console.log(`✅ Created ${sampleReplies.length} sample replies`);
    }

    async writeAllTables() {
        console.log('\n💾 Writing tables to JSON files...');
        
        for (const [tableName, data] of Object.entries(this.tables)) {
            const filePath = path.join(this.dataDir, `${tableName}.json`);
            await fs.writeFile(filePath, JSON.stringify(data, null, 2));
            console.log(`✅ Wrote ${tableName}.json (${data.length} records)`);
        }
    }

    printSummary() {
        console.log('\n📊 Database Summary');
        console.log('==================');
        
        Object.entries(this.tables).forEach(([tableName, data]) => {
            console.log(`📄 ${tableName}: ${data.length} records`);
        });
        
        console.log('\n🚀 Next Steps:');
        console.log('1. Run the conversion script: ./run_conversion.sh');
        console.log('2. Configure environment variables');
        console.log('3. Test the support desk locally');
        console.log('4. Deploy to Netlify');
        
        console.log('\n🔐 Default Credentials:');
        console.log('Admin: admin@knirv.com / admin123');
        console.log('Customer: customer@example.com / customer123');
        
        console.log('\n📁 Data Location:');
        console.log(`${this.dataDir}/`);
        console.log('├── users.json');
        console.log('├── tickets.json');
        console.log('├── replies.json');
        console.log('├── attachments.json');
        console.log('├── categories.json');
        console.log('└── settings.json');
    }
}

// Main execution
if (require.main === module) {
    // Check if already initialized first
    checkIfInitialized().then(() => {
        // Check if bcrypt is available
        try {
            require('bcryptjs');
        } catch (error) {
            console.error('❌ bcryptjs is required but not installed.');
            console.error('Please run: npm install bcryptjs');
            process.exit(1);
        }

        const initializer = new DatabaseInitializer();
        initializer.initialize().then(async () => {
            // Create initialization marker
            await fs.writeFile(INIT_MARKER_FILE, new Date().toISOString());
        }).catch(console.error);
    });
}

module.exports = DatabaseInitializer;
