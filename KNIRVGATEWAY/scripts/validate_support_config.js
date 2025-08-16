#!/usr/bin/env node

/**
 * Configuration Validation Script
 * 
 * This script validates the configuration setup for the HESK support desk
 * and provides recommendations for security and performance.
 */

const fs = require('fs');
const path = require('path');

class ConfigValidator {
    constructor() {
        this.functionsDir = path.join(__dirname, '..', 'netlify', 'functions');
        this.configPath = path.join(this.functionsDir, '.config');
        this.configExamplePath = path.join(this.functionsDir, '.config.example');
        this.configLoaderPath = path.join(this.functionsDir, 'config-loader.js');
        
        this.results = {
            passed: 0,
            failed: 0,
            warnings: 0,
            checks: []
        };
    }

    async validate() {
        console.log('⚙️  Configuration Validation');
        console.log('============================');
        console.log('');

        // Run validation checks
        await this.checkConfigFiles();
        await this.checkConfigContent();
        await this.checkSecurity();
        await this.checkDatabaseConfig();
        await this.checkEmailConfig();
        await this.checkFileUploadConfig();

        // Print results
        this.printResults();
        
        return this.results.failed === 0;
    }

    check(name, condition, message, type = 'error') {
        const status = condition ? 'PASS' : 'FAIL';
        const icon = condition ? '✅' : (type === 'warning' ? '⚠️' : '❌');
        
        console.log(`${icon} ${name}: ${status}`);
        if (!condition && message) {
            console.log(`   ${message}`);
        }
        
        this.results.checks.push({
            name,
            status,
            message: condition ? null : message,
            type
        });
        
        if (condition) {
            this.results.passed++;
        } else {
            if (type === 'warning') {
                this.results.warnings++;
            } else {
                this.results.failed++;
            }
        }
        
        return condition;
    }

    async checkConfigFiles() {
        console.log('📁 Checking Configuration Files...');
        
        this.check(
            'config-loader.js exists',
            fs.existsSync(this.configLoaderPath),
            'Missing config-loader.js - run conversion script'
        );
        
        this.check(
            '.config.example exists',
            fs.existsSync(this.configExamplePath),
            'Missing .config.example template'
        );
        
        this.check(
            '.config file exists',
            fs.existsSync(this.configPath),
            'Create .config file from .config.example'
        );
    }

    async checkConfigContent() {
        console.log('\n📋 Checking Configuration Content...');
        
        if (!fs.existsSync(this.configPath)) {
            this.check('Configuration content', false, 'Cannot check content - .config file missing');
            return;
        }
        
        const configContent = fs.readFileSync(this.configPath, 'utf8');
        
        // Check for required fields
        const requiredFields = ['DB_TYPE', 'JWT_SECRET'];
        for (const field of requiredFields) {
            this.check(
                `${field} is set`,
                configContent.includes(`${field}=`),
                `Missing required field: ${field}`
            );
        }
        
        // Check for recommended fields
        const recommendedFields = ['COMPANY_NAME', 'SUPPORT_DESK_TITLE', 'ADMIN_EMAIL'];
        for (const field of recommendedFields) {
            this.check(
                `${field} is configured`,
                configContent.includes(`${field}=`),
                `Recommended field not set: ${field}`,
                'warning'
            );
        }
    }

    async checkSecurity() {
        console.log('\n🔒 Checking Security Configuration...');
        
        if (!fs.existsSync(this.configPath)) {
            return;
        }
        
        const configContent = fs.readFileSync(this.configPath, 'utf8');
        
        // Check JWT secret
        const jwtSecretMatch = configContent.match(/JWT_SECRET=(.+)/);
        if (jwtSecretMatch) {
            const jwtSecret = jwtSecretMatch[1].trim().replace(/['"]/g, '');
            
            this.check(
                'JWT_SECRET is not default',
                !jwtSecret.includes('your-') && !jwtSecret.includes('change-this'),
                'JWT_SECRET should be changed from default value'
            );
            
            this.check(
                'JWT_SECRET is long enough',
                jwtSecret.length >= 32,
                'JWT_SECRET should be at least 32 characters long',
                'warning'
            );
            
            this.check(
                'JWT_SECRET is complex',
                /[A-Z]/.test(jwtSecret) && /[a-z]/.test(jwtSecret) && /[0-9]/.test(jwtSecret),
                'JWT_SECRET should contain uppercase, lowercase, and numbers',
                'warning'
            );
        }
        
        // Check password requirements
        const minPasswordLength = configContent.match(/MIN_PASSWORD_LENGTH=(\d+)/);
        if (minPasswordLength) {
            const length = parseInt(minPasswordLength[1]);
            this.check(
                'Password length is secure',
                length >= 8,
                'MIN_PASSWORD_LENGTH should be at least 8',
                'warning'
            );
        }
        
        // Check debug mode
        const debugMode = configContent.match(/DEBUG=(true|false)/);
        if (debugMode) {
            this.check(
                'Debug mode is disabled in production',
                debugMode[1] === 'false',
                'DEBUG should be false in production',
                'warning'
            );
        }
    }

    async checkDatabaseConfig() {
        console.log('\n🗄️  Checking Database Configuration...');
        
        if (!fs.existsSync(this.configPath)) {
            return;
        }
        
        const configContent = fs.readFileSync(this.configPath, 'utf8');
        
        const dbTypeMatch = configContent.match(/DB_TYPE=(.+)/);
        if (dbTypeMatch) {
            const dbType = dbTypeMatch[1].trim().replace(/['"]/g, '');
            
            this.check(
                'DB_TYPE is valid',
                ['json', 'supabase'].includes(dbType),
                'DB_TYPE must be either "json" or "supabase"'
            );
            
            if (dbType === 'supabase') {
                this.check(
                    'SUPABASE_URL is set',
                    configContent.includes('SUPABASE_URL=') && !configContent.includes('your-project'),
                    'SUPABASE_URL must be configured for Supabase database'
                );
                
                this.check(
                    'SUPABASE_ANON_KEY is set',
                    configContent.includes('SUPABASE_ANON_KEY=') && !configContent.includes('your-anon-key'),
                    'SUPABASE_ANON_KEY must be configured for Supabase database'
                );
            }
            
            if (dbType === 'json') {
                const dataDir = path.join(__dirname, '..', '..', 'data');
                this.check(
                    'JSON database directory exists',
                    fs.existsSync(dataDir),
                    'Run "node init_database.js" to initialize JSON database',
                    'warning'
                );
            }
        }
    }

    async checkEmailConfig() {
        console.log('\n📧 Checking Email Configuration...');
        
        if (!fs.existsSync(this.configPath)) {
            return;
        }
        
        const configContent = fs.readFileSync(this.configPath, 'utf8');
        
        const emailEnabled = configContent.includes('ENABLE_EMAIL_NOTIFICATIONS=true');
        
        if (emailEnabled) {
            const requiredEmailFields = ['SMTP_HOST', 'SMTP_PORT', 'SMTP_USER', 'SMTP_PASS'];
            for (const field of requiredEmailFields) {
                this.check(
                    `${field} is configured`,
                    configContent.includes(`${field}=`) && !configContent.includes('your-'),
                    `${field} must be configured when email notifications are enabled`
                );
            }
            
            this.check(
                'EMAIL_FROM is set',
                configContent.includes('EMAIL_FROM=') && !configContent.includes('your'),
                'EMAIL_FROM should be configured for email notifications',
                'warning'
            );
        } else {
            this.check(
                'Email notifications status',
                true,
                'Email notifications are disabled',
                'warning'
            );
        }
    }

    async checkFileUploadConfig() {
        console.log('\n📎 Checking File Upload Configuration...');
        
        if (!fs.existsSync(this.configPath)) {
            return;
        }
        
        const configContent = fs.readFileSync(this.configPath, 'utf8');
        
        const maxFileSizeMatch = configContent.match(/MAX_FILE_SIZE=(\d+)/);
        if (maxFileSizeMatch) {
            const maxSize = parseInt(maxFileSizeMatch[1]);
            this.check(
                'File size limit is reasonable',
                maxSize <= 50 * 1024 * 1024, // 50MB
                'MAX_FILE_SIZE is very large - consider security implications',
                'warning'
            );
        }
        
        const maxFilesMatch = configContent.match(/MAX_FILES_PER_TICKET=(\d+)/);
        if (maxFilesMatch) {
            const maxFiles = parseInt(maxFilesMatch[1]);
            this.check(
                'File count limit is reasonable',
                maxFiles <= 10,
                'MAX_FILES_PER_TICKET is high - consider performance implications',
                'warning'
            );
        }
        
        this.check(
            'Allowed file types are configured',
            configContent.includes('ALLOWED_FILE_TYPES='),
            'ALLOWED_FILE_TYPES should be configured for security',
            'warning'
        );
    }

    printResults() {
        console.log('\n📊 Configuration Validation Summary');
        console.log('===================================');
        console.log(`✅ Passed: ${this.results.passed}`);
        console.log(`❌ Failed: ${this.results.failed}`);
        console.log(`⚠️  Warnings: ${this.results.warnings}`);
        console.log('');
        
        if (this.results.failed === 0) {
            console.log('🎉 Configuration validation passed!');
            
            if (this.results.warnings > 0) {
                console.log('');
                console.log('⚠️  Please review the warnings above for optimal security and performance.');
            }
            
            console.log('');
            console.log('✅ Your configuration is ready for deployment!');
        } else {
            console.log('❌ Configuration validation failed. Please fix the issues above.');
            console.log('');
            console.log('💡 Quick fixes:');
            console.log('1. Copy .config.example to .config');
            console.log('2. Generate a secure JWT_SECRET');
            console.log('3. Configure your database settings');
            console.log('4. Set your company information');
        }
    }
}

// Main execution
if (require.main === module) {
    const validator = new ConfigValidator();
    validator.validate().then(success => {
        process.exit(success ? 0 : 1);
    }).catch(error => {
        console.error('Validation error:', error);
        process.exit(1);
    });
}

module.exports = ConfigValidator;
