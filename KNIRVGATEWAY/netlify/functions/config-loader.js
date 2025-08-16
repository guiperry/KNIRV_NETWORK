// Configuration Loader for HESK Netlify Functions
// This module loads configuration from .config file with fallback to environment variables

const fs = require('fs');
const path = require('path');

class ConfigLoader {
    constructor() {
        this.config = {};
        this.configPath = path.join(__dirname, '.config');
        this.loaded = false;
    }

    loadConfig() {
        if (this.loaded) {
            return this.config;
        }

        // Try to load from .config file first
        try {
            if (fs.existsSync(this.configPath)) {
                const configContent = fs.readFileSync(this.configPath, 'utf8');
                this.parseConfigFile(configContent);
                console.log('✅ Configuration loaded from .config file');
            } else {
                console.log('⚠️  .config file not found, using environment variables only');
            }
        } catch (error) {
            console.error('❌ Error loading .config file:', error.message);
            console.log('📋 Falling back to environment variables');
        }

        // Merge with environment variables (env vars take precedence)
        this.mergeWithEnvironment();
        
        // Set defaults
        this.setDefaults();
        
        this.loaded = true;
        return this.config;
    }

    parseConfigFile(content) {
        const lines = content.split('\n');
        
        for (const line of lines) {
            const trimmedLine = line.trim();
            
            // Skip comments and empty lines
            if (trimmedLine.startsWith('#') || trimmedLine === '') {
                continue;
            }
            
            // Parse key=value pairs
            const equalIndex = trimmedLine.indexOf('=');
            if (equalIndex > 0) {
                const key = trimmedLine.substring(0, equalIndex).trim();
                let value = trimmedLine.substring(equalIndex + 1).trim();
                
                // Remove quotes if present
                if ((value.startsWith('"') && value.endsWith('"')) || 
                    (value.startsWith("'") && value.endsWith("'"))) {
                    value = value.slice(1, -1);
                }
                
                this.config[key] = value;
            }
        }
    }

    mergeWithEnvironment() {
        // Environment variables override config file values
        const envVars = [
            'DB_TYPE',
            'SUPABASE_URL',
            'SUPABASE_ANON_KEY',
            'SUPABASE_SERVICE_ROLE_KEY',
            'JWT_SECRET',
            'SMTP_HOST',
            'SMTP_PORT',
            'SMTP_USER',
            'SMTP_PASS',
            'EMAIL_FROM',
            'EMAIL_FROM_NAME',
            'MAX_FILE_SIZE',
            'MAX_FILES_PER_TICKET',
            'ALLOWED_FILE_TYPES',
            'FILE_STORAGE',
            'LOCAL_UPLOAD_PATH',
            'AWS_ACCESS_KEY_ID',
            'AWS_SECRET_ACCESS_KEY',
            'AWS_REGION',
            'AWS_S3_BUCKET',
            'CLOUDINARY_CLOUD_NAME',
            'CLOUDINARY_API_KEY',
            'CLOUDINARY_API_SECRET',
            'RATE_LIMIT_REQUESTS',
            'RATE_LIMIT_WINDOW',
            'CORS_ORIGINS',
            'ALLOW_REGISTRATION',
            'MIN_PASSWORD_LENGTH',
            'REQUIRE_SPECIAL_CHARS',
            'ENABLE_EMAIL_NOTIFICATIONS',
            'ADMIN_EMAIL',
            'NOTIFY_NEW_TICKET',
            'NOTIFY_TICKET_REPLY',
            'NOTIFY_TICKET_STATUS_CHANGE',
            'LOG_LEVEL',
            'LOG_REQUESTS',
            'LOG_FILE_PATH',
            'NODE_ENV',
            'DEBUG',
            'DETAILED_ERRORS',
            'DB_CONNECTION_LIMIT',
            'DB_ACQUIRE_TIMEOUT',
            'DB_TIMEOUT',
            'ENABLE_CACHE',
            'CACHE_TTL',
            'CACHE_PROVIDER',
            'REDIS_URL',
            'REDIS_PASSWORD',
            'GA_TRACKING_ID',
            'ENABLE_ANALYTICS',
            'ENABLE_BACKUPS',
            'BACKUP_FREQUENCY',
            'BACKUP_RETENTION',
            'BACKUP_STORAGE',
            'BACKUP_PATH',
            'SLACK_WEBHOOK_URL',
            'DISCORD_WEBHOOK_URL',
            'ENABLE_SLACK_NOTIFICATIONS',
            'ENABLE_DISCORD_NOTIFICATIONS',
            'COMPANY_NAME',
            'SUPPORT_DESK_TITLE',
            'DEFAULT_PRIORITY',
            'DEFAULT_CATEGORY',
            'AUTO_CLOSE_DAYS',
            'ENABLE_SATISFACTION_SURVEYS',
            'CUSTOM_CSS_URL',
            'CUSTOM_JS_URL',
            'MAINTENANCE_MODE',
            'MAINTENANCE_MESSAGE',
            'MAINTENANCE_END_TIME'
        ];

        for (const envVar of envVars) {
            if (process.env[envVar] !== undefined) {
                this.config[envVar] = process.env[envVar];
            }
        }
    }

    setDefaults() {
        const defaults = {
            DB_TYPE: 'json',
            JWT_SECRET: 'your-jwt-secret-change-this-in-production',
            MAX_FILE_SIZE: '10485760',
            MAX_FILES_PER_TICKET: '5',
            ALLOWED_FILE_TYPES: 'image/jpeg,image/png,image/gif,text/plain,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document',
            FILE_STORAGE: 'local',
            LOCAL_UPLOAD_PATH: '../../uploads',
            RATE_LIMIT_REQUESTS: '60',
            RATE_LIMIT_WINDOW: '60000',
            CORS_ORIGINS: '*',
            ALLOW_REGISTRATION: 'true',
            MIN_PASSWORD_LENGTH: '8',
            REQUIRE_SPECIAL_CHARS: 'false',
            ENABLE_EMAIL_NOTIFICATIONS: 'false',
            ADMIN_EMAIL: 'admin@knirv.com',
            NOTIFY_NEW_TICKET: 'true',
            NOTIFY_TICKET_REPLY: 'true',
            NOTIFY_TICKET_STATUS_CHANGE: 'true',
            LOG_LEVEL: 'info',
            LOG_REQUESTS: 'false',
            NODE_ENV: 'production',
            DEBUG: 'false',
            DETAILED_ERRORS: 'false',
            DB_CONNECTION_LIMIT: '10',
            DB_ACQUIRE_TIMEOUT: '60000',
            DB_TIMEOUT: '60000',
            ENABLE_CACHE: 'false',
            CACHE_TTL: '300',
            CACHE_PROVIDER: 'memory',
            ENABLE_ANALYTICS: 'false',
            ENABLE_BACKUPS: 'false',
            BACKUP_FREQUENCY: '24',
            BACKUP_RETENTION: '30',
            BACKUP_STORAGE: 'local',
            BACKUP_PATH: './backups',
            ENABLE_SLACK_NOTIFICATIONS: 'false',
            ENABLE_DISCORD_NOTIFICATIONS: 'false',
            COMPANY_NAME: 'KNIRV Network',
            SUPPORT_DESK_TITLE: 'KNIRV Support Desk',
            DEFAULT_PRIORITY: 'medium',
            DEFAULT_CATEGORY: 'general',
            AUTO_CLOSE_DAYS: '30',
            ENABLE_SATISFACTION_SURVEYS: 'false',
            MAINTENANCE_MODE: 'false',
            MAINTENANCE_MESSAGE: 'The support desk is currently under maintenance. Please try again later.'
        };

        for (const [key, value] of Object.entries(defaults)) {
            if (this.config[key] === undefined) {
                this.config[key] = value;
            }
        }
    }

    get(key, defaultValue = undefined) {
        if (!this.loaded) {
            this.loadConfig();
        }
        return this.config[key] !== undefined ? this.config[key] : defaultValue;
    }

    getBoolean(key, defaultValue = false) {
        const value = this.get(key);
        if (value === undefined) return defaultValue;
        return value === 'true' || value === '1' || value === 'yes' || value === 'on';
    }

    getNumber(key, defaultValue = 0) {
        const value = this.get(key);
        if (value === undefined) return defaultValue;
        const num = parseInt(value, 10);
        return isNaN(num) ? defaultValue : num;
    }

    getArray(key, defaultValue = [], separator = ',') {
        const value = this.get(key);
        if (value === undefined) return defaultValue;
        return value.split(separator).map(item => item.trim()).filter(item => item.length > 0);
    }

    isDevelopment() {
        return this.get('NODE_ENV') === 'development';
    }

    isProduction() {
        return this.get('NODE_ENV') === 'production';
    }

    isDebugEnabled() {
        return this.getBoolean('DEBUG');
    }

    isMaintenanceMode() {
        return this.getBoolean('MAINTENANCE_MODE');
    }

    validate() {
        const errors = [];
        
        // Required fields validation
        const requiredFields = ['JWT_SECRET'];
        
        for (const field of requiredFields) {
            if (!this.get(field) || this.get(field) === 'your-jwt-secret-change-this-in-production') {
                errors.push(`${field} must be set to a secure value`);
            }
        }
        
        // Database type validation
        const dbType = this.get('DB_TYPE');
        if (!['json', 'supabase'].includes(dbType)) {
            errors.push('DB_TYPE must be either "json" or "supabase"');
        }
        
        // Supabase validation
        if (dbType === 'supabase') {
            if (!this.get('SUPABASE_URL')) {
                errors.push('SUPABASE_URL is required when DB_TYPE is "supabase"');
            }
            if (!this.get('SUPABASE_ANON_KEY')) {
                errors.push('SUPABASE_ANON_KEY is required when DB_TYPE is "supabase"');
            }
        }
        
        // Email validation
        if (this.getBoolean('ENABLE_EMAIL_NOTIFICATIONS')) {
            const requiredEmailFields = ['SMTP_HOST', 'SMTP_PORT', 'SMTP_USER', 'SMTP_PASS'];
            for (const field of requiredEmailFields) {
                if (!this.get(field)) {
                    errors.push(`${field} is required when email notifications are enabled`);
                }
            }
        }
        
        return errors;
    }

    printConfig() {
        if (!this.loaded) {
            this.loadConfig();
        }
        
        console.log('📋 Current Configuration:');
        console.log('========================');
        console.log(`Database Type: ${this.get('DB_TYPE')}`);
        console.log(`Environment: ${this.get('NODE_ENV')}`);
        console.log(`Debug Mode: ${this.getBoolean('DEBUG')}`);
        console.log(`Maintenance Mode: ${this.getBoolean('MAINTENANCE_MODE')}`);
        console.log(`Email Notifications: ${this.getBoolean('ENABLE_EMAIL_NOTIFICATIONS')}`);
        console.log(`Max File Size: ${this.getNumber('MAX_FILE_SIZE')} bytes`);
        console.log(`Company: ${this.get('COMPANY_NAME')}`);
        
        const errors = this.validate();
        if (errors.length > 0) {
            console.log('\n⚠️  Configuration Warnings:');
            errors.forEach(error => console.log(`   - ${error}`));
        }
    }
}

// Create singleton instance
const config = new ConfigLoader();

module.exports = config;
