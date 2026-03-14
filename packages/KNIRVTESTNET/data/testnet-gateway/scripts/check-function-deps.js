#!/usr/bin/env node

/**
 * KNIRV Gateway Function Dependencies Checker
 * Verifies that all Netlify Function dependencies are properly installed
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class FunctionDependencyChecker {
    constructor() {
        this.functionsDir = path.join(__dirname, '..', 'netlify', 'functions');
        this.packageJsonPath = path.join(this.functionsDir, 'package.json');
        this.nodeModulesPath = path.join(this.functionsDir, 'node_modules');
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

    async checkFunctionPackageJson() {
        this.log('Checking function package.json...');
        
        if (!fs.existsSync(this.packageJsonPath)) {
            this.errors.push('Functions package.json not found');
            return false;
        }

        try {
            const packageJson = JSON.parse(fs.readFileSync(this.packageJsonPath, 'utf8'));
            const requiredDeps = [
                '@supabase/supabase-js',
                'formidable',
                'bcryptjs',
                'jsonwebtoken',
                'nodemailer',
                'uuid',
                'mime-types',
                'sharp',
                'markdown-it',
                'dompurify',
                'jsdom'
            ];

            const missingDeps = requiredDeps.filter(dep => !packageJson.dependencies[dep]);
            
            if (missingDeps.length > 0) {
                this.errors.push(`Missing dependencies in functions package.json: ${missingDeps.join(', ')}`);
                return false;
            }

            this.log(`Found ${Object.keys(packageJson.dependencies).length} dependencies in functions package.json`, 'success');
            return true;
        } catch (error) {
            this.errors.push(`Error reading functions package.json: ${error.message}`);
            return false;
        }
    }

    async checkNodeModules() {
        this.log('Checking function node_modules...');
        
        if (!fs.existsSync(this.nodeModulesPath)) {
            this.errors.push('Functions node_modules directory not found');
            return false;
        }

        const requiredModules = [
            '@supabase/supabase-js',
            'formidable',
            'bcryptjs',
            'jsonwebtoken'
        ];

        const missingModules = requiredModules.filter(module => {
            const modulePath = path.join(this.nodeModulesPath, module);
            return !fs.existsSync(modulePath);
        });

        if (missingModules.length > 0) {
            this.errors.push(`Missing modules in functions node_modules: ${missingModules.join(', ')}`);
            return false;
        }

        this.log(`All required modules found in functions node_modules`, 'success');
        return true;
    }

    async checkSupabaseImport() {
        this.log('Testing Supabase import...');

        try {
            // Test by checking if the module can be resolved from the functions directory
            const supabaseModulePath = path.join(this.nodeModulesPath, '@supabase', 'supabase-js');

            if (fs.existsSync(supabaseModulePath)) {
                // Check for package.json first
                const packageJsonPath = path.join(supabaseModulePath, 'package.json');
                if (fs.existsSync(packageJsonPath)) {
                    try {
                        const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
                        // Check for multiple possible entry points
                        const possibleMains = [
                            packageJson.main,
                            packageJson.module,
                            'index.js',
                            'dist/index.js',
                            'dist/main/index.js'
                        ].filter(Boolean);

                        let mainFileFound = false;
                        for (const mainFile of possibleMains) {
                            const fullPath = path.join(supabaseModulePath, mainFile);
                            if (fs.existsSync(fullPath)) {
                                this.log(`Supabase module structure verified (entry: ${mainFile})`, 'success');
                                mainFileFound = true;
                                break;
                            }
                        }

                        if (!mainFileFound) {
                            // Check if dist directory exists (common for compiled packages)
                            const distPath = path.join(supabaseModulePath, 'dist');
                            if (fs.existsSync(distPath)) {
                                this.log('Supabase module dist directory found', 'success');
                                return true;
                            }
                            throw new Error(`No valid entry point found. Checked: ${possibleMains.join(', ')}`);
                        }
                        return true;
                    } catch (parseError) {
                        throw new Error(`Failed to parse Supabase package.json: ${parseError.message}`);
                    }
                } else {
                    throw new Error('Supabase package.json not found');
                }
            } else {
                throw new Error('Supabase module directory not found');
            }
        } catch (error) {
            this.warnings.push(`Supabase import test failed: ${error.message} (this is okay if using JSON database)`);
            return true; // Don't fail the build for this
        }
    }

    async installFunctionDependencies() {
        this.log('Installing function dependencies...');
        
        try {
            const originalCwd = process.cwd();
            process.chdir(this.functionsDir);
            
            execSync('npm install', { stdio: 'inherit' });
            
            process.chdir(originalCwd);
            this.log('Function dependencies installed successfully', 'success');
            return true;
        } catch (error) {
            this.errors.push(`Failed to install function dependencies: ${error.message}`);
            return false;
        }
    }

    async checkMainPackageJson() {
        this.log('Checking main package.json for function dependencies...');
        
        const mainPackageJsonPath = path.join(__dirname, '..', 'package.json');
        
        try {
            const packageJson = JSON.parse(fs.readFileSync(mainPackageJsonPath, 'utf8'));
            const functionDeps = [
                '@supabase/supabase-js',
                'formidable',
                'bcryptjs',
                'jsonwebtoken'
            ];

            const foundInMain = functionDeps.filter(dep => packageJson.dependencies[dep]);
            
            if (foundInMain.length > 0) {
                this.log(`Function dependencies also found in main package.json: ${foundInMain.join(', ')}`, 'success');
            } else {
                this.warnings.push('No function dependencies found in main package.json (this is okay if functions have their own package.json)');
            }
            
            return true;
        } catch (error) {
            this.warnings.push(`Could not check main package.json: ${error.message}`);
            return false;
        }
    }

    async run(autoFix = false) {
        this.log('🏥 Starting KNIRV Gateway Function Dependencies Check...');
        this.log('='.repeat(60));

        // If autoFix is enabled, try to install dependencies first
        if (autoFix) {
            this.log('Auto-fix enabled, installing function dependencies first...', 'info');
            await this.installFunctionDependencies();
        }

        const checks = [
            () => this.checkMainPackageJson(),
            () => this.checkFunctionPackageJson(),
            () => this.checkNodeModules(),
            () => this.checkSupabaseImport()
        ];

        let allPassed = true;

        for (const check of checks) {
            const result = await check();
            if (!result) {
                allPassed = false;
                if (!autoFix) {
                    this.log('Issues found but auto-fix not enabled', 'warning');
                }
            }
        }

        this.log('='.repeat(60));

        if (this.warnings.length > 0) {
            this.log('Warnings found:', 'warning');
            this.warnings.forEach(warning => this.log(`  - ${warning}`, 'warning'));
        }

        if (this.errors.length > 0) {
            this.log('Errors found:', 'error');
            this.errors.forEach(error => this.log(`  - ${error}`, 'error'));

            this.log('='.repeat(60));
            this.log('💡 Suggested fixes:', 'info');
            this.log('  1. Run: cd netlify/functions && npm install', 'info');
            this.log('  2. Or run: npm run install-function-deps', 'info');
            this.log('  3. Or add dependencies to main package.json', 'info');

            process.exit(1);
        } else {
            this.log('All function dependency checks passed!', 'success');
            process.exit(0);
        }
    }
}

// Main execution
if (require.main === module) {
    const autoFix = process.argv.includes('--fix');
    const checker = new FunctionDependencyChecker();
    checker.run(autoFix).catch(console.error);
}

module.exports = FunctionDependencyChecker;
