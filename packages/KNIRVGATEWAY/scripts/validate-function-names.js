#!/usr/bin/env node

/**
 * KNIRV Gateway Function Name Validator
 * Validates that all Netlify Function names comply with naming requirements
 * (only alphanumeric characters, hyphens, and underscores)
 */

const fs = require('fs');
const path = require('path');

class FunctionNameValidator {
    constructor() {
        this.functionsDir = path.join(__dirname, '..', 'netlify', 'functions');
        this.errors = [];
        this.warnings = [];
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            info: '🔍',
            success: '✅',
            warning: '⚠️',
            error: '❌'
        }[type] || '📝';
        
        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    isValidFunctionName(filename) {
        // Remove .js extension for validation
        const name = filename.replace(/\.js$/, '');
        
        // Check if name contains only alphanumeric characters, hyphens, and underscores
        const validPattern = /^[a-zA-Z0-9_-]+$/;
        return validPattern.test(name);
    }

    suggestValidName(filename) {
        // Remove .js extension
        const name = filename.replace(/\.js$/, '');
        
        // Replace invalid characters with hyphens
        const suggested = name
            .replace(/[^a-zA-Z0-9_-]/g, '-')
            .replace(/-+/g, '-')  // Replace multiple hyphens with single hyphen
            .replace(/^-|-$/g, ''); // Remove leading/trailing hyphens
        
        return suggested + '.js';
    }

    async validateFunctionNames() {
        this.log('🔍 Validating Netlify Function names...');

        if (!fs.existsSync(this.functionsDir)) {
            this.warnings.push('Functions directory not found');
            return true;
        }

        const files = fs.readdirSync(this.functionsDir)
            .filter(file => file.endsWith('.js') && fs.statSync(path.join(this.functionsDir, file)).isFile());

        let invalidFiles = [];

        for (const file of files) {
            if (!this.isValidFunctionName(file)) {
                const suggested = this.suggestValidName(file);
                invalidFiles.push({
                    current: file,
                    suggested: suggested
                });
            }
        }

        if (invalidFiles.length > 0) {
            this.errors.push(`Found ${invalidFiles.length} function(s) with invalid names`);
            
            this.log('Invalid function names found:', 'error');
            invalidFiles.forEach(({ current, suggested }) => {
                this.log(`  ${current} → ${suggested}`, 'error');
            });

            this.log('', 'info');
            this.log('To fix these issues, run:', 'info');
            invalidFiles.forEach(({ current, suggested }) => {
                this.log(`  mv "network-website/netlify/functions/${current}" "network-website/netlify/functions/${suggested}"`, 'info');
            });

            return false;
        }

        this.log(`All ${files.length} function names are valid`, 'success');
        return true;
    }

    async run() {
        this.log('🚀 Starting Function Name Validation...');
        this.log('='.repeat(50));

        const success = await this.validateFunctionNames();

        this.log('='.repeat(50));
        
        if (this.warnings.length > 0) {
            this.log('Warnings found:', 'warning');
            this.warnings.forEach(warning => this.log(`  - ${warning}`, 'warning'));
        }

        if (this.errors.length > 0) {
            this.log('Errors found:', 'error');
            this.errors.forEach(error => this.log(`  - ${error}`, 'error'));
            
            this.log('', 'info');
            this.log('💡 Function names must contain only:', 'info');
            this.log('  - Alphanumeric characters (a-z, A-Z, 0-9)', 'info');
            this.log('  - Hyphens (-)', 'info');
            this.log('  - Underscores (_)', 'info');
            
            process.exit(1);
        } else {
            this.log('All function name validations passed!', 'success');
            process.exit(0);
        }
    }
}

// Main execution
if (require.main === module) {
    const validator = new FunctionNameValidator();
    validator.run().catch(console.error);
}

module.exports = FunctionNameValidator;
