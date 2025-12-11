#!/usr/bin/env node

/**
 * Test script to verify Netlify dependency patches are working
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

class NetlifyPatchTester {
    constructor() {
        this.projectRoot = path.resolve(__dirname, '..');
        this.functionsDir = path.join(this.projectRoot, 'netlify', 'functions');
        this.nodeModulesDir = path.join(this.functionsDir, 'node_modules');
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            info: '🔍',
            success: '✅',
            warning: '⚠️',
            error: '❌'
        }[type] || 'ℹ️';
        
        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    fileExists(filePath) {
        try {
            return fs.statSync(filePath).isFile();
        } catch {
            return false;
        }
    }

    testBcryptjsPatch() {
        this.log('Testing bcryptjs patch...');
        
        const distFile = path.join(this.nodeModulesDir, 'bcryptjs', 'dist', 'bcrypt.js');
        
        if (this.fileExists(distFile)) {
            this.log('bcryptjs dist/bcrypt.js exists', 'success');
            
            // Test that the file has content
            const content = fs.readFileSync(distFile, 'utf8');
            if (content.length > 100) {
                this.log('bcryptjs dist file has content', 'success');
                return true;
            } else {
                this.log('bcryptjs dist file is empty or too small', 'error');
                return false;
            }
        } else {
            this.log('bcryptjs dist/bcrypt.js missing', 'error');
            return false;
        }
    }

    testFormidablePatch() {
        this.log('Testing formidable patch...');
        
        const requiredFiles = [
            'index.js',
            'Formidable.js',
            'PersistentFile.js',
            'VolatileFile.js',
            'FormidableError.js'
        ];

        let allGood = true;
        
        for (const file of requiredFiles) {
            const filePath = path.join(this.nodeModulesDir, 'formidable', 'dist', file);
            
            if (this.fileExists(filePath)) {
                this.log(`formidable dist/${file} exists`, 'success');
            } else {
                this.log(`formidable dist/${file} missing`, 'error');
                allGood = false;
            }
        }

        // Test subdirectories
        const subdirs = ['parsers', 'plugins', 'helpers'];
        for (const subdir of subdirs) {
            const subdirPath = path.join(this.nodeModulesDir, 'formidable', 'dist', subdir);
            try {
                if (fs.statSync(subdirPath).isDirectory()) {
                    this.log(`formidable dist/${subdir}/ directory exists`, 'success');
                } else {
                    this.log(`formidable dist/${subdir}/ not a directory`, 'error');
                    allGood = false;
                }
            } catch {
                this.log(`formidable dist/${subdir}/ directory missing`, 'error');
                allGood = false;
            }
        }

        return allGood;
    }

    testNetlifyBuild() {
        this.log('Testing Netlify CLI build simulation...');
        
        try {
            // Simulate what Netlify CLI does - try to require the modules
            const bcryptjsPath = path.join(this.nodeModulesDir, 'bcryptjs');
            const formidablePath = path.join(this.nodeModulesDir, 'formidable');
            
            // Test bcryptjs require
            try {
                const bcryptjsIndex = path.join(bcryptjsPath, 'index.js');
                const bcryptjsContent = fs.readFileSync(bcryptjsIndex, 'utf8');
                if (bcryptjsContent.includes('./dist/bcrypt.js')) {
                    const distPath = path.join(bcryptjsPath, 'dist', 'bcrypt.js');
                    if (this.fileExists(distPath)) {
                        this.log('bcryptjs require path will work', 'success');
                    } else {
                        this.log('bcryptjs require path will fail - dist file missing', 'error');
                        return false;
                    }
                }
            } catch (error) {
                this.log(`bcryptjs require test failed: ${error.message}`, 'error');
                return false;
            }

            // Test formidable require
            try {
                const formidablePackage = path.join(formidablePath, 'package.json');
                const packageContent = JSON.parse(fs.readFileSync(formidablePackage, 'utf8'));
                const mainPath = packageContent.exports?.['.']?.require?.default || packageContent.main;
                
                if (mainPath && mainPath.includes('./dist/')) {
                    const fullPath = path.join(formidablePath, mainPath);
                    if (this.fileExists(fullPath)) {
                        this.log('formidable require path will work', 'success');
                    } else {
                        this.log(`formidable require path will fail - ${mainPath} missing`, 'error');
                        return false;
                    }
                }
            } catch (error) {
                this.log(`formidable require test failed: ${error.message}`, 'error');
                return false;
            }

            this.log('Netlify build simulation passed', 'success');
            return true;
        } catch (error) {
            this.log(`Netlify build simulation failed: ${error.message}`, 'error');
            return false;
        }
    }

    async run() {
        this.log('🧪 Starting Netlify patches test...');
        this.log(`Testing in: ${this.functionsDir}`);

        if (!fs.existsSync(this.nodeModulesDir)) {
            this.log('node_modules not found - run npm install first', 'error');
            return false;
        }

        const bcryptjsOk = this.testBcryptjsPatch();
        const formidableOk = this.testFormidablePatch();
        const buildOk = this.testNetlifyBuild();

        const allPassed = bcryptjsOk && formidableOk && buildOk;

        this.log('\n📊 Test Results:');
        this.log(`bcryptjs patch: ${bcryptjsOk ? '✅ PASS' : '❌ FAIL'}`);
        this.log(`formidable patch: ${formidableOk ? '✅ PASS' : '❌ FAIL'}`);
        this.log(`build simulation: ${buildOk ? '✅ PASS' : '❌ FAIL'}`);

        this.log(`\n${allPassed ? '🎉 All tests passed!' : '⚠️ Some tests failed'}`, allPassed ? 'success' : 'error');
        
        return allPassed;
    }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
    const tester = new NetlifyPatchTester();
    tester.run()
        .then(success => {
            process.exit(success ? 0 : 1);
        })
        .catch(error => {
            console.error('❌ Test failed with error:', error);
            process.exit(1);
        });
}

export { NetlifyPatchTester };
