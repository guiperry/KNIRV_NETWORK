#!/usr/bin/env node

/**
 * Discourse Database Initialization Script
 * 
 * This script initializes the JSON database with sample data and creates
 * the necessary data structure for the Discourse forum.
 */

const fs = require('fs').promises;
const path = require('path');
const bcrypt = require('bcryptjs');

class DiscourseDBInitializer {
    constructor() {
        this.dataDir = path.join(__dirname, '..', 'forum', 'data');
        this.tables = {
            users: [],
            topics: [],
            posts: [],
            categories: [],
            groups: [],
            group_users: [],
            badges: [],
            user_badges: [],
            tags: [],
            topic_tags: [],
            notifications: [],
            user_actions: [],
            site_settings: [],
            uploads: []
        };
    }

    async initialize() {
        console.log('🗄️  Initializing Discourse JSON Database...');
        console.log('==========================================');

        try {
            // Create data directory
            await this.createDataDirectory();
            
            // Initialize tables with sample data
            await this.initializeUsers();
            await this.initializeCategories();
            await this.initializeGroups();
            await this.initializeBadges();
            await this.initializeTags();
            await this.initializeSiteSettings();
            await this.initializeSampleContent();
            
            // Write all tables to JSON files
            await this.writeAllTables();
            
            console.log('\n✅ Discourse database initialization completed successfully!');
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
            password_hash: adminPassword,
            name: 'KNIRV Administrator',
            avatar_url: '/img/avatars/admin.png',
            bio: 'Forum administrator for KNIRV Network',
            location: 'KNIRV Network',
            website: 'https://knirv.com',
            trust_level: 4,
            admin: true,
            moderator: true,
            suspended: false,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            last_seen_at: new Date().toISOString()
        };

        // Create moderator user
        const modPassword = await bcrypt.hash('mod123', 10);
        const modUser = {
            id: 2,
            username: 'moderator',
            email: 'mod@knirv.com',
            password_hash: modPassword,
            name: 'Forum Moderator',
            avatar_url: '/img/avatars/moderator.png',
            bio: 'Community moderator',
            trust_level: 3,
            admin: false,
            moderator: true,
            suspended: false,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            last_seen_at: new Date().toISOString()
        };

        // Create sample users
        const users = [adminUser, modUser];
        
        for (let i = 3; i <= 10; i++) {
            const userPassword = await bcrypt.hash('user123', 10);
            users.push({
                id: i,
                username: `user${i}`,
                email: `user${i}@example.com`,
                password_hash: userPassword,
                name: `Sample User ${i}`,
                avatar_url: `/img/avatars/user${i}.png`,
                bio: `Sample user ${i} for testing`,
                trust_level: Math.floor(Math.random() * 3),
                admin: false,
                moderator: false,
                suspended: false,
                created_at: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000).toISOString(),
                updated_at: new Date().toISOString(),
                last_seen_at: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString()
            });
        }

        this.tables.users = users;
        console.log(`✅ Created ${users.length} users`);
        console.log('   Admin: admin@knirv.com / admin123');
        console.log('   Moderator: mod@knirv.com / mod123');
        console.log('   Sample users: user3@example.com / user123 (etc.)');
    }

    async initializeCategories() {
        console.log('\n📂 Initializing categories...');
        
        const categories = [
            {
                id: 1,
                name: 'General Discussion',
                slug: 'general',
                description: 'General discussions about KNIRV Network',
                color: '#3498db',
                text_color: '#ffffff',
                parent_category_id: null,
                topic_count: 0,
                post_count: 0,
                position: 1,
                read_restricted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 2,
                name: 'Announcements',
                slug: 'announcements',
                description: 'Official announcements from the KNIRV team',
                color: '#e74c3c',
                text_color: '#ffffff',
                parent_category_id: null,
                topic_count: 0,
                post_count: 0,
                position: 2,
                read_restricted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 3,
                name: 'Technical Support',
                slug: 'support',
                description: 'Get help with technical issues',
                color: '#f39c12',
                text_color: '#ffffff',
                parent_category_id: null,
                topic_count: 0,
                post_count: 0,
                position: 3,
                read_restricted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 4,
                name: 'Development',
                slug: 'development',
                description: 'Discussions about KNIRV development',
                color: '#2ecc71',
                text_color: '#ffffff',
                parent_category_id: null,
                topic_count: 0,
                post_count: 0,
                position: 4,
                read_restricted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 5,
                name: 'Feature Requests',
                slug: 'features',
                description: 'Suggest new features for KNIRV',
                color: '#9b59b6',
                text_color: '#ffffff',
                parent_category_id: null,
                topic_count: 0,
                post_count: 0,
                position: 5,
                read_restricted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 6,
                name: 'Off Topic',
                slug: 'off-topic',
                description: 'Discussions not related to KNIRV',
                color: '#95a5a6',
                text_color: '#ffffff',
                parent_category_id: null,
                topic_count: 0,
                post_count: 0,
                position: 6,
                read_restricted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            }
        ];

        this.tables.categories = categories;
        console.log(`✅ Created ${categories.length} categories`);
    }

    async initializeGroups() {
        console.log('\n👥 Initializing groups...');
        
        const groups = [
            {
                id: 1,
                name: 'admins',
                display_name: 'Administrators',
                description: 'Forum administrators',
                user_count: 1,
                mentionable_level: 0,
                messageable_level: 0,
                visibility_level: 0,
                automatic: true,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 2,
                name: 'moderators',
                display_name: 'Moderators',
                description: 'Forum moderators',
                user_count: 1,
                mentionable_level: 0,
                messageable_level: 0,
                visibility_level: 0,
                automatic: true,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 3,
                name: 'trust_level_0',
                display_name: 'New Users',
                description: 'New users (trust level 0)',
                user_count: 0,
                mentionable_level: 99,
                messageable_level: 99,
                visibility_level: 0,
                automatic: true,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 4,
                name: 'developers',
                display_name: 'Developers',
                description: 'KNIRV developers and contributors',
                user_count: 0,
                mentionable_level: 0,
                messageable_level: 0,
                visibility_level: 0,
                automatic: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            }
        ];

        // Create group memberships
        const groupUsers = [
            { id: 1, group_id: 1, user_id: 1, owner: true, notification_level: 3, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 2, group_id: 2, user_id: 2, owner: true, notification_level: 3, created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
        ];

        this.tables.groups = groups;
        this.tables.group_users = groupUsers;
        console.log(`✅ Created ${groups.length} groups and ${groupUsers.length} memberships`);
    }

    async initializeBadges() {
        console.log('\n🏆 Initializing badges...');
        
        const badges = [
            {
                id: 1,
                name: 'Welcome',
                description: 'Completed the new user tutorial',
                badge_type_id: 3,
                grant_count: 0,
                icon: 'fa-certificate',
                listable: true,
                target_posts: false,
                enabled: true,
                auto_revoke: false,
                badge_grouping_id: 1,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 2,
                name: 'First Post',
                description: 'Made your first post',
                badge_type_id: 3,
                grant_count: 0,
                icon: 'fa-pencil',
                listable: true,
                target_posts: false,
                enabled: true,
                auto_revoke: false,
                badge_grouping_id: 1,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 3,
                name: 'Regular',
                description: 'Promoted to regular user status',
                badge_type_id: 2,
                grant_count: 0,
                icon: 'fa-user',
                listable: true,
                target_posts: false,
                enabled: true,
                auto_revoke: false,
                badge_grouping_id: 2,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            }
        ];

        this.tables.badges = badges;
        console.log(`✅ Created ${badges.length} badges`);
    }

    async initializeTags() {
        console.log('\n🏷️  Initializing tags...');
        
        const tags = [
            { id: 1, name: 'knirv', topic_count: 0, pm_topic_count: 0, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 2, name: 'blockchain', topic_count: 0, pm_topic_count: 0, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 3, name: 'network', topic_count: 0, pm_topic_count: 0, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 4, name: 'development', topic_count: 0, pm_topic_count: 0, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 5, name: 'support', topic_count: 0, pm_topic_count: 0, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 6, name: 'feature-request', topic_count: 0, pm_topic_count: 0, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 7, name: 'bug', topic_count: 0, pm_topic_count: 0, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 8, name: 'tutorial', topic_count: 0, pm_topic_count: 0, created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
        ];

        this.tables.tags = tags;
        console.log(`✅ Created ${tags.length} tags`);
    }

    async initializeSiteSettings() {
        console.log('\n⚙️  Initializing site settings...');
        
        const settings = [
            { id: 1, name: 'title', data_type: 1, value: 'KNIRV Forum', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 2, name: 'site_description', data_type: 1, value: 'KNIRV Network Community Forum', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 3, name: 'contact_email', data_type: 1, value: 'admin@knirv.com', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 4, name: 'company_name', data_type: 1, value: 'KNIRV Network', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 5, name: 'logo_url', data_type: 1, value: '/img/logo.png', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 6, name: 'favicon_url', data_type: 1, value: '/img/favicon.png', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 7, name: 'allow_user_registration', data_type: 5, value: 'true', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 8, name: 'min_password_length', data_type: 3, value: '8', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 9, name: 'topics_per_page', data_type: 3, value: '30', created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
            { id: 10, name: 'posts_per_page', data_type: 3, value: '20', created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
        ];

        this.tables.site_settings = settings;
        console.log(`✅ Created ${settings.length} site settings`);
    }

    async initializeSampleContent() {
        console.log('\n📝 Creating sample content...');
        
        // Create welcome topic
        const welcomeTopic = {
            id: 1,
            title: 'Welcome to KNIRV Forum!',
            slug: 'welcome-to-knirv-forum',
            category_id: 1,
            user_id: 1,
            views: 0,
            posts_count: 1,
            reply_count: 0,
            like_count: 0,
            pinned: true,
            closed: false,
            archived: false,
            visible: true,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            last_posted_at: new Date().toISOString()
        };

        const welcomePost = {
            id: 1,
            topic_id: 1,
            user_id: 1,
            post_number: 1,
            raw: 'Welcome to the KNIRV Network Community Forum!\n\nThis is your space to discuss all things related to KNIRV, ask questions, share ideas, and connect with other community members.\n\n## Getting Started\n\n- **Read the guidelines** in the pinned topics\n- **Introduce yourself** in the General Discussion category\n- **Ask questions** in the Technical Support category\n- **Share your ideas** in the Feature Requests category\n\nWe\'re excited to have you here!',
            cooked: '<p>Welcome to the KNIRV Network Community Forum!</p><p>This is your space to discuss all things related to KNIRV, ask questions, share ideas, and connect with other community members.</p><h2>Getting Started</h2><ul><li><strong>Read the guidelines</strong> in the pinned topics</li><li><strong>Introduce yourself</strong> in the General Discussion category</li><li><strong>Ask questions</strong> in the Technical Support category</li><li><strong>Share your ideas</strong> in the Feature Requests category</li></ul><p>We\'re excited to have you here!</p>',
            reply_to_post_number: null,
            like_count: 0,
            reply_count: 0,
            quote_count: 0,
            deleted_at: null,
            hidden: false,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString()
        };

        this.tables.topics = [welcomeTopic];
        this.tables.posts = [welcomePost];
        
        console.log('✅ Created sample welcome topic and post');
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
        console.log('1. Run the conversion script: node convert_discourse_to_netlify.js');
        console.log('2. Configure environment variables');
        console.log('3. Test the forum locally');
        console.log('4. Deploy to Netlify');
        
        console.log('\n🔐 Default Credentials:');
        console.log('Admin: admin@knirv.com / admin123');
        console.log('Moderator: mod@knirv.com / mod123');
        console.log('Sample Users: user3@example.com / user123 (etc.)');
        
        console.log('\n📁 Data Location:');
        console.log(`${this.dataDir}/`);
        console.log('├── users.json');
        console.log('├── topics.json');
        console.log('├── posts.json');
        console.log('├── categories.json');
        console.log('├── groups.json');
        console.log('├── group_users.json');
        console.log('├── badges.json');
        console.log('├── user_badges.json');
        console.log('├── tags.json');
        console.log('├── topic_tags.json');
        console.log('├── notifications.json');
        console.log('├── user_actions.json');
        console.log('├── site_settings.json');
        console.log('└── uploads.json');
    }
}

// Main execution
if (require.main === module) {
    // Check if bcrypt is available
    try {
        require('bcryptjs');
    } catch (error) {
        console.error('❌ bcryptjs is required but not installed.');
        console.error('Please run: npm install bcryptjs');
        process.exit(1);
    }
    
    const initializer = new DiscourseDBInitializer();
    initializer.initialize().catch(console.error);
}

module.exports = DiscourseDBInitializer;
