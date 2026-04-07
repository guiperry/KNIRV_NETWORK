#!/usr/bin/env node

/**
 * KNIRV Gateway - Reset Initialization Markers
 * Removes initialization markers to allow re-initialization of apps
 */

const fs = require('fs');
const path = require('path');

const markers = [
    path.join(__dirname, '..', 'forum', 'data', '.initialized'),
    path.join(__dirname, '..', 'support-desk', 'data', '.initialized')
];

console.log('🔄 Resetting initialization markers...');

markers.forEach(marker => {
    if (fs.existsSync(marker)) {
        fs.unlinkSync(marker);
        console.log(`✅ Removed: ${path.relative(process.cwd(), marker)}`);
    } else {
        console.log(`⚠️  Not found: ${path.relative(process.cwd(), marker)}`);
    }
});

console.log('🎉 Initialization markers reset. Apps will re-initialize on next build.');
