#!/usr/bin/env node

/**
 * KNIRV NEXUS Frontend Health Checker
 * Checks NEXUS frontend build integrity for unified architecture
 */

const fs = require('fs');
const path = require('path');

class NexusHealthChecker {
    constructor() {
        this.issues = [];
        this.warnings = [];
    }

    log(message, type = 'info') {
        const prefix = {
            'info': '🔍',
            'warn': '⚠️ ',
            'error': '❌',
            'success': '✅',
            'fix': '🔧'
        }[type] || 'ℹ️ ';

        console.log(`${prefix} [NEXUS] ${message}`);
    }

    async checkNexusDirectory() {
        this.log('Checking NEXUS frontend directory structure...');

        // In unified architecture, NEXUS frontend is in data/knirvnexus/portal
        const nexusPath = path.join(process.cwd(), '../../data/knirvnexus/portal');
        const requiredFiles = [
            'package.json',
            '.next',
            'public',
            'server.js'
        ];

        if (!fs.existsSync(nexusPath)) {
            this.log('NEXUS frontend directory does not exist', 'warn');
            this.warnings.push('NEXUS frontend directory missing - run build-nexus-frontend.sh');
            return false;
        }

        let missingFiles = [];
        for (const file of requiredFiles) {
            const filePath = path.join(nexusPath, file);
            if (!fs.existsSync(filePath)) {
                missingFiles.push(file);
            }
        }

        if (missingFiles.length > 0) {
            this.log(`Missing files: ${missingFiles.join(', ')}`, 'warn');
            this.warnings.push(`Missing NEXUS frontend files: ${missingFiles.join(', ')}`);
        } else {
            this.log('All required files present', 'success');
        }

        return missingFiles.length === 0;
    }

    async checkNexusDependencies() {
        this.log('Checking NEXUS portal dependencies...');

        const nexusPath = path.join(process.cwd(), 'nexus-portal');

        if (!fs.existsSync(nexusPath)) {
            this.warnings.push('nexus-portal directory missing - dependencies check skipped');
            return false;
        }

        const nodeModulesPath = path.join(nexusPath, 'node_modules');
        const packageLockPath = path.join(nexusPath, 'package-lock.json');

        if (!fs.existsSync(nodeModulesPath)) {
            this.warnings.push('nexus-portal node_modules missing - will install');
            return false;
        }

        if (!fs.existsSync(packageLockPath)) {
            this.warnings.push('nexus-portal package-lock.json missing');
        }

        // Check for critical dependencies
        const criticalDeps = ['react', 'vite', 'typescript'];
        let missingDeps = [];

        for (const dep of criticalDeps) {
            const depPath = path.join(nodeModulesPath, dep);
            if (!fs.existsSync(depPath)) {
                missingDeps.push(dep);
                this.issues.push(`Critical dependency missing: ${dep}`);
            }
        }
        
        return this.issues.length === 0;
    }

    async checkNexusBuild() {
        this.log('Checking NEXUS portal build artifacts...');
        
        const nexusPath = path.join(process.cwd(), 'nexus-portal');
        const distPath = path.join(nexusPath, 'dist');
        const indexPath = path.join(distPath, 'index.html');
        const assetsPath = path.join(distPath, 'assets');
        
        if (!fs.existsSync(distPath)) {
            this.warnings.push('nexus-portal dist directory missing - needs build');
            return false;
        }
        
        if (!fs.existsSync(indexPath)) {
            this.issues.push('nexus-portal index.html missing from dist');
            return false;
        }
        
        if (!fs.existsSync(assetsPath)) {
            this.issues.push('nexus-portal assets directory missing from dist');
            return false;
        }
        
        // Check for CSS and JS files
        try {
            const assetFiles = fs.readdirSync(assetsPath);
            const hasCss = assetFiles.some(file => file.endsWith('.css'));
            const hasJs = assetFiles.some(file => file.endsWith('.js'));
            
            if (!hasCss) {
                this.issues.push('No CSS files found in nexus-portal build');
            }
            
            if (!hasJs) {
                this.issues.push('No JavaScript files found in nexus-portal build');
            }
            
            this.log(`Found ${assetFiles.length} asset files`, 'success');
            
        } catch (error) {
            this.issues.push('Could not read nexus-portal assets directory');
            return false;
        }
        
        return true;
    }

    async checkBuildFreshness() {
        this.log('Checking NEXUS portal build freshness...');
        
        const nexusPath = path.join(process.cwd(), 'nexus-portal');
        const distPath = path.join(nexusPath, 'dist');
        const indexPath = path.join(distPath, 'index.html');
        const srcPath = path.join(nexusPath, 'src');
        
        if (!fs.existsSync(indexPath)) {
            return false; // Already handled in checkNexusBuild
        }
        
        try {
            const buildTime = fs.statSync(indexPath).mtime.getTime();
            
            // Check if any source files are newer than the build
            const checkSourceFiles = (dir) => {
                const files = fs.readdirSync(dir);
                for (const file of files) {
                    const filePath = path.join(dir, file);
                    const stat = fs.statSync(filePath);
                    
                    if (stat.isDirectory()) {
                        if (checkSourceFiles(filePath)) return true;
                    } else if (stat.mtime.getTime() > buildTime) {
                        this.warnings.push(`Source file ${file} is newer than build`);
                        return true;
                    }
                }
                return false;
            };
            
            if (fs.existsSync(srcPath)) {
                checkSourceFiles(srcPath);
            }
            
            // Check build age
            const ageMs = Date.now() - buildTime;
            const ageMinutes = Math.floor(ageMs / (1000 * 60));
            
            if (ageMinutes > 120) { // 2 hours
                this.warnings.push(`NEXUS portal build is ${ageMinutes} minutes old`);
            } else {
                this.log(`Build is ${ageMinutes} minutes old`, 'success');
            }
            
        } catch (error) {
            this.warnings.push('Could not check build freshness');
        }
        
        return true;
    }

    async testViteBuild() {
        this.log('Testing Vite build capability...');
        
        const nexusPath = path.join(process.cwd(), 'nexus-portal');
        
        try {
            // Test if vite can be invoked
            const result = execSync('npm run build-only --dry-run', {
                cwd: nexusPath,
                encoding: 'utf8',
                timeout: 10000,
                stdio: 'pipe'
            });
            
            this.log('Vite build test passed', 'success');
            return true;
            
        } catch (error) {
            this.issues.push(`Vite build test failed: ${error.message}`);
            return false;
        }
    }

    async repairNexusPortal() {
        this.log('🔧 Starting NEXUS portal repair process...');

        const nexusPath = path.join(process.cwd(), 'nexus-portal');

        // Step 1: Create nexus-portal directory if missing
        if (!fs.existsSync(nexusPath)) {
            this.log('Creating nexus-portal directory...', 'fix');

            // Try to copy from production KNIRVGATEWAY
            const productionNexusPath = path.join(process.cwd(), '../../../KNIRVGATEWAY/nexus-portal');

            if (fs.existsSync(productionNexusPath)) {
                this.log('Copying nexus-portal from production KNIRVGATEWAY...', 'fix');
                try {
                    execSync(`cp -r "${productionNexusPath}" "${nexusPath}"`, { stdio: 'pipe' });
                    this.log('Successfully copied nexus-portal from production', 'success');
                } catch (error) {
                    this.log(`Failed to copy from production: ${error.message}`, 'error');
                    return false;
                }
            } else {
                this.log('Production nexus-portal not found, creating minimal structure...', 'fix');
                fs.mkdirSync(nexusPath, { recursive: true });
                fs.mkdirSync(path.join(nexusPath, 'src'), { recursive: true });

                // Create minimal package.json
                const minimalPackageJson = {
                    "name": "nexus-portal-testnet",
                    "version": "1.0.0",
                    "type": "module",
                    "scripts": {
                        "dev": "vite",
                        "build": "tsc && vite build",
                        "preview": "vite preview"
                    },
                    "dependencies": {
                        "react": "^18.2.0",
                        "react-dom": "^18.2.0"
                    },
                    "devDependencies": {
                        "@types/react": "^18.2.0",
                        "@types/react-dom": "^18.2.0",
                        "@vitejs/plugin-react": "^4.0.0",
                        "typescript": "^5.0.0",
                        "vite": "^4.4.0"
                    }
                };

                fs.writeFileSync(
                    path.join(nexusPath, 'package.json'),
                    JSON.stringify(minimalPackageJson, null, 2)
                );
            }
        }

        // Step 2: Install/repair dependencies
        if (fs.existsSync(nexusPath)) {
            this.log('Installing/repairing nexus-portal dependencies...', 'fix');

            try {
                const originalCwd = process.cwd();
                process.chdir(nexusPath);

                // Clean install
                if (fs.existsSync('node_modules')) {
                    execSync('rm -rf node_modules package-lock.json', { stdio: 'pipe' });
                }

                execSync('npm install', { stdio: 'pipe' });
                this.log('Successfully installed nexus-portal dependencies', 'success');

                // Step 3: Build the project if package.json has build script
                try {
                    const packageJson = JSON.parse(fs.readFileSync('package.json', 'utf8'));
                    if (packageJson.scripts && packageJson.scripts.build) {
                        this.log('Building nexus-portal...', 'fix');
                        execSync('npm run build', { stdio: 'pipe' });
                        this.log('Successfully built nexus-portal', 'success');
                    }
                } catch (buildError) {
                    this.log(`Build failed (non-critical): ${buildError.message}`, 'warn');
                }

                // Return to original directory
                process.chdir(originalCwd);

            } catch (error) {
                this.log(`Failed to install dependencies: ${error.message}`, 'error');
                process.chdir(originalCwd);
                return false;
            }
        }

        this.log('NEXUS portal repair completed', 'success');
        return true;
    }

    async runNexusHealthCheck() {
        this.log('🏥 Starting NEXUS portal health check...');
        
        const checks = [
            this.checkNexusDirectory(),
            this.checkNexusDependencies(),
            this.checkNexusBuild(),
            this.checkBuildFreshness()
        ];
        
        await Promise.all(checks);
        
        // Report results
        if (this.issues.length === 0 && this.warnings.length === 0) {
            this.log('NEXUS portal health check passed! 🎉', 'success');
            return true;
        }
        
        if (this.warnings.length > 0) {
            this.log(`Found ${this.warnings.length} warnings:`, 'warn');
            this.warnings.forEach(warning => this.log(`  - ${warning}`, 'warn'));
        }
        
        if (this.issues.length > 0) {
            this.log(`Found ${this.issues.length} critical issues:`, 'error');
            this.issues.forEach(issue => this.log(`  - ${issue}`, 'error'));
            
            this.log('NEXUS portal health check failed', 'error');
            return false;
        }
        
        // Only warnings, still considered passing
        return true;
    }
}

// Run health check if called directly
if (require.main === module) {
    const checker = new NexusHealthChecker();

    // Check for repair flag
    const shouldRepair = process.argv.includes('--repair') || process.argv.includes('--fix');

    if (shouldRepair) {
        console.log('🔧 Running NEXUS portal repair...');
        checker.repairNexusPortal()
            .then(success => {
                if (success) {
                    console.log('✅ Repair completed, running health check...');
                    return checker.runNexusHealthCheck();
                } else {
                    console.log('❌ Repair failed');
                    return false;
                }
            })
            .then(success => {
                process.exit(success ? 0 : 1);
            })
            .catch(error => {
                console.error('❌ NEXUS repair/health check failed with error:', error.message);
                process.exit(1);
            });
    } else {
        checker.runNexusHealthCheck()
            .then(success => {
                process.exit(success ? 0 : 1);
            })
            .catch(error => {
                console.error('❌ NEXUS health check failed with error:', error.message);
                process.exit(1);
            });
    }
}

module.exports = NexusHealthChecker;
