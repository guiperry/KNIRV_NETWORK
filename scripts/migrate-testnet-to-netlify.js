#!/usr/bin/env node

/**
 * DEPRECATED: KNIRV Testnet to Netlify Migration Script
 * 
 * This script has been replaced with a comprehensive deployment solution.
 * Use the new deployment commands instead:
 * 
 * - make deploy-testnet                 # Deploy complete testnet (AWS + Netlify)
 * - make deploy-testnet-infrastructure  # Deploy AWS infrastructure only
 * - make deploy-testnet-services       # Deploy Docker services only
 * - make update-testnet-frontend       # Update Netlify frontend only
 * 
 * Access the live testnet at: https://knirv.com/testnet
 */

console.log('🧪 DEPRECATED: Use make deploy-testnet instead');
console.log('');
console.log('This script has been replaced with a comprehensive deployment solution:');
console.log('');
console.log('📋 Available Commands:');
console.log('  make deploy-testnet                 # Deploy complete testnet (AWS + Netlify)');
console.log('  make deploy-testnet-infrastructure  # Deploy AWS infrastructure only');
console.log('  make deploy-testnet-services       # Deploy Docker services only');
console.log('  make update-testnet-frontend       # Update Netlify frontend only');
console.log('');
console.log('🌐 Access: https://knirv.com/testnet');
console.log('');
console.log('For more information, see:');
console.log('  - README.md');
console.log('  - KNIRVTESTNET/README.md');
console.log('  - deployment/ansible/environments/testnet.yml');
console.log('');

// Exit with deprecation notice
process.exit(0);
