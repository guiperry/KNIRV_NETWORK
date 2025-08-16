#!/usr/bin/env node

/**
 * Discourse Configuration Validator
 * 
 * This script validates the Discourse forum configuration to ensure
 * all required settings are properly configured before deployment.
 */

const fs = require('fs');
const path = require('path');

class DiscourseConfigValidator {
    constructor() {
        this.configPath = path.join(__dirname, '..', 'netlify', 'functions', '.config');
        this.examplePath = path.join(__dirname, '.config.example');
        this.errors = [];
        this.warnings = [];
        this.config = {};
    }

    async validate() {
        console.log('🔍 Validating Discourse Configuration...');
        console.log('=====================================');

        try {
            // Check if config file exists
            await this.checkConfigFile();
            
            // Load configuration
            await this.loadConfiguration();
            
            // Validate required settings
            await this.validateRequired();
            
            // Validate database configuration
            await this.validateDatabase();
            
            // Validate authentication settings
            await this.validateAuthentication();
            
            // Validate email configuration
            await this.validateEmail();
            
            // Validate file upload settings
            await this.validateFileUpload();
            
            // Validate security settings
            await this.validateSecurity();
            
            // Validate performance settings
            await this.validatePerformance();
            
            // Print results
            this.printResults();
            
        } catch (error) {
            console.error('\n❌ Validation failed:', error.message);
            process.exit(1);
        }
    }

    async checkConfigFile() {
        if (!fs.existsSync(this.configPath)) {
            this.errors.push('Configuration file not found at netlify/functions/.config');
            
            if (fs.existsSync(this.examplePath)) {
                console.log('💡 Copy .config.example to netlify/functions/.config and configure your settings');
                console.log(`   cp ${this.examplePath} ${this.configPath}`);
            }
            
            throw new Error('Configuration file missing');
        }
        
        console.log('✅ Configuration file found');
    }

    async loadConfiguration() {
        try {
            const configContent = fs.readFileSync(this.configPath, 'utf8');
            const lines = configContent.split('\n');
            
            for (const line of lines) {
                const trimmed = line.trim();
                if (trimmed && !trimmed.startsWith('#') && trimmed.includes('=')) {
                    const [key, ...valueParts] = trimmed.split('=');
                    const value = valueParts.join('=').trim();
                    this.config[key.trim()] = value;
                }
            }
            
            console.log(`✅ Loaded ${Object.keys(this.config).length} configuration settings`);
            
        } catch (error) {
            throw new Error(`Failed to load configuration: ${error.message}`);
        }
    }

    validateRequired() {
        console.log('\n🔧 Validating required settings...');
        
        const required = [
            'DB_TYPE',
            'JWT_SECRET',
            'FORUM_TITLE',
            'ADMIN_EMAIL'
        ];
        
        for (const key of required) {
            if (!this.config[key] || this.config[key].trim() === '') {
                this.errors.push(`Required setting missing: ${key}`);
            }
        }
        
        // Check JWT secret strength
        if (this.config.JWT_SECRET) {
            if (this.config.JWT_SECRET.length < 32) {
                this.errors.push('JWT_SECRET must be at least 32 characters long');
            }
            
            if (this.config.JWT_SECRET === 'your-super-secret-jwt-key-here-make-it-long-and-random') {
                this.errors.push('JWT_SECRET must be changed from the default value');
            }
        }
        
        console.log('✅ Required settings validation completed');
    }

    validateDatabase() {
        console.log('\n🗄️  Validating database configuration...');
        
        const dbType = this.config.DB_TYPE;
        
        if (!dbType) {
            this.errors.push('DB_TYPE must be specified (json or supabase)');
            return;
        }
        
        if (dbType !== 'json' && dbType !== 'supabase') {
            this.errors.push('DB_TYPE must be either "json" or "supabase"');
        }
        
        if (dbType === 'supabase') {
            const required = ['SUPABASE_URL', 'SUPABASE_ANON_KEY'];
            for (const key of required) {
                if (!this.config[key]) {
                    this.errors.push(`${key} is required when using Supabase`);
                }
            }
            
            if (this.config.SUPABASE_URL && !this.config.SUPABASE_URL.startsWith('https://')) {
                this.errors.push('SUPABASE_URL must be a valid HTTPS URL');
            }
        }
        
        if (dbType === 'json') {
            // Check if data directory exists
            const dataDir = path.join(__dirname, '..', '..', 'data');
            if (!fs.existsSync(dataDir)) {
                this.warnings.push('Data directory does not exist. Run init_discourse_database.js first.');
            }
        }
        
        console.log('✅ Database configuration validation completed');
    }

    validateAuthentication() {
        console.log('\n🔐 Validating authentication settings...');
        
        // JWT expiration
        if (this.config.JWT_EXPIRATION) {
            const expiration = parseInt(this.config.JWT_EXPIRATION);
            if (isNaN(expiration) || expiration < 3600) {
                this.warnings.push('JWT_EXPIRATION should be at least 3600 seconds (1 hour)');
            }
            if (expiration > 604800) {
                this.warnings.push('JWT_EXPIRATION longer than 7 days may be a security risk');
            }
        }
        
        // Password requirements
        if (this.config.MIN_PASSWORD_LENGTH) {
            const minLength = parseInt(this.config.MIN_PASSWORD_LENGTH);
            if (isNaN(minLength) || minLength < 8) {
                this.warnings.push('MIN_PASSWORD_LENGTH should be at least 8 characters');
            }
        }
        
        console.log('✅ Authentication settings validation completed');
    }

    validateEmail() {
        console.log('\n📧 Validating email configuration...');
        
        if (this.config.ENABLE_EMAIL_NOTIFICATIONS === 'true') {
            const required = ['SMTP_HOST', 'SMTP_PORT', 'SMTP_USER', 'SMTP_PASS', 'EMAIL_FROM'];
            for (const key of required) {
                if (!this.config[key]) {
                    this.warnings.push(`${key} is required when email notifications are enabled`);
                }
            }
            
            // Validate email format
            if (this.config.EMAIL_FROM && !this.isValidEmail(this.config.EMAIL_FROM)) {
                this.errors.push('EMAIL_FROM must be a valid email address');
            }
            
            if (this.config.ADMIN_EMAIL && !this.isValidEmail(this.config.ADMIN_EMAIL)) {
                this.errors.push('ADMIN_EMAIL must be a valid email address');
            }
            
            // Check SMTP port
            if (this.config.SMTP_PORT) {
                const port = parseInt(this.config.SMTP_PORT);
                if (isNaN(port) || port < 1 || port > 65535) {
                    this.errors.push('SMTP_PORT must be a valid port number (1-65535)');
                }
            }
        }
        
        console.log('✅ Email configuration validation completed');
    }

    validateFileUpload() {
        console.log('\n📁 Validating file upload settings...');
        
        // File size validation
        if (this.config.MAX_FILE_SIZE) {
            const maxSize = parseInt(this.config.MAX_FILE_SIZE);
            if (isNaN(maxSize) || maxSize < 1024) {
                this.warnings.push('MAX_FILE_SIZE should be at least 1024 bytes');
            }
            if (maxSize > 50 * 1024 * 1024) {
                this.warnings.push('MAX_FILE_SIZE larger than 50MB may cause performance issues');
            }
        }
        
        // File storage validation
        const fileStorage = this.config.FILE_STORAGE;
        if (fileStorage) {
            if (!['local', 'aws-s3', 'cloudinary'].includes(fileStorage)) {
                this.errors.push('FILE_STORAGE must be one of: local, aws-s3, cloudinary');
            }
            
            if (fileStorage === 'aws-s3') {
                const required = ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY', 'AWS_S3_BUCKET'];
                for (const key of required) {
                    if (!this.config[key]) {
                        this.errors.push(`${key} is required when using AWS S3 storage`);
                    }
                }
            }
            
            if (fileStorage === 'cloudinary') {
                const required = ['CLOUDINARY_CLOUD_NAME', 'CLOUDINARY_API_KEY', 'CLOUDINARY_API_SECRET'];
                for (const key of required) {
                    if (!this.config[key]) {
                        this.errors.push(`${key} is required when using Cloudinary storage`);
                    }
                }
            }
        }
        
        console.log('✅ File upload settings validation completed');
    }

    validateSecurity() {
        console.log('\n🛡️  Validating security settings...');
        
        // Rate limiting
        if (this.config.RATE_LIMIT_REQUESTS) {
            const rateLimit = parseInt(this.config.RATE_LIMIT_REQUESTS);
            if (isNaN(rateLimit) || rateLimit < 1) {
                this.warnings.push('RATE_LIMIT_REQUESTS should be a positive number');
            }
            if (rateLimit > 1000) {
                this.warnings.push('RATE_LIMIT_REQUESTS higher than 1000 may not provide effective protection');
            }
        }
        
        // CORS origins
        if (this.config.CORS_ORIGINS) {
            const origins = this.config.CORS_ORIGINS.split(',');
            for (const origin of origins) {
                const trimmed = origin.trim();
                if (trimmed && !trimmed.startsWith('http://') && !trimmed.startsWith('https://')) {
                    this.warnings.push(`CORS origin should include protocol: ${trimmed}`);
                }
            }
        }
        
        // Check for development settings in production
        if (this.config.NODE_ENV === 'production') {
            if (this.config.DEBUG === 'true') {
                this.warnings.push('DEBUG mode should be disabled in production');
            }
            if (this.config.DETAILED_ERRORS === 'true') {
                this.warnings.push('DETAILED_ERRORS should be disabled in production');
            }
        }
        
        console.log('✅ Security settings validation completed');
    }

    validatePerformance() {
        console.log('\n⚡ Validating performance settings...');
        
        // Pagination settings
        const paginationSettings = ['TOPICS_PER_PAGE', 'POSTS_PER_PAGE', 'SEARCH_RESULTS_PER_PAGE'];
        for (const setting of paginationSettings) {
            if (this.config[setting]) {
                const value = parseInt(this.config[setting]);
                if (isNaN(value) || value < 1) {
                    this.warnings.push(`${setting} should be a positive number`);
                }
                if (value > 100) {
                    this.warnings.push(`${setting} higher than 100 may impact performance`);
                }
            }
        }
        
        // Cache settings
        if (this.config.ENABLE_CACHE === 'true' && this.config.CACHE_TTL) {
            const cacheTtl = parseInt(this.config.CACHE_TTL);
            if (isNaN(cacheTtl) || cacheTtl < 60) {
                this.warnings.push('CACHE_TTL should be at least 60 seconds');
            }
        }
        
        console.log('✅ Performance settings validation completed');
    }

    isValidEmail(email) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    }

    printResults() {
        console.log('\n📊 Validation Results');
        console.log('====================');
        
        if (this.errors.length === 0 && this.warnings.length === 0) {
            console.log('✅ All validations passed! Your configuration looks good.');
        } else {
            if (this.errors.length > 0) {
                console.log('\n❌ Errors (must be fixed):');
                this.errors.forEach((error, index) => {
                    console.log(`   ${index + 1}. ${error}`);
                });
            }
            
            if (this.warnings.length > 0) {
                console.log('\n⚠️  Warnings (recommended to fix):');
                this.warnings.forEach((warning, index) => {
                    console.log(`   ${index + 1}. ${warning}`);
                });
            }
        }
        
        console.log('\n📋 Configuration Summary:');
        console.log(`   Database Type: ${this.config.DB_TYPE || 'Not set'}`);
        console.log(`   Forum Title: ${this.config.FORUM_TITLE || 'Not set'}`);
        console.log(`   Admin Email: ${this.config.ADMIN_EMAIL || 'Not set'}`);
        console.log(`   Email Notifications: ${this.config.ENABLE_EMAIL_NOTIFICATIONS || 'false'}`);
        console.log(`   File Storage: ${this.config.FILE_STORAGE || 'local'}`);
        console.log(`   Environment: ${this.config.NODE_ENV || 'development'}`);
        
        if (this.errors.length > 0) {
            console.log('\n🚨 Please fix the errors above before deploying to production.');
            process.exit(1);
        } else {
            console.log('\n🎉 Configuration validation completed successfully!');
            
            if (this.warnings.length > 0) {
                console.log('💡 Consider addressing the warnings for optimal performance and security.');
            }
        }
    }
}

// Main execution
if (require.main === module) {
    const validator = new DiscourseConfigValidator();
    validator.validate().catch(console.error);
}

module.exports = DiscourseConfigValidator;
