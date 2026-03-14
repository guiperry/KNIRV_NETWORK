#!/usr/bin/env node

/**
 * Debug script to check Render.com port configuration
 */

console.log('🔍 RENDER PORT DEBUG INFORMATION');
console.log('================================');
console.log('');

console.log('Environment Variables:');
console.log('- NODE_ENV:', process.env.NODE_ENV || 'not set');
console.log('- PORT:', process.env.PORT || 'not set');
console.log('- RENDER:', process.env.RENDER || 'not set');
console.log('- RENDER_SERVICE_ID:', process.env.RENDER_SERVICE_ID || 'not set');
console.log('- RENDER_EXTERNAL_URL:', process.env.RENDER_EXTERNAL_URL || 'not set');
console.log('- RENDER_SERVICE_NAME:', process.env.RENDER_SERVICE_NAME || 'not set');
console.log('');

console.log('Process Information:');
console.log('- Node.js Version:', process.version);
console.log('- Platform:', process.platform);
console.log('- Architecture:', process.arch);
console.log('- Working Directory:', process.cwd());
console.log('');

console.log('Network Information:');
const os = require('os');
const networkInterfaces = os.networkInterfaces();
Object.keys(networkInterfaces).forEach(interfaceName => {
  const interfaces = networkInterfaces[interfaceName];
  interfaces.forEach(interface => {
    if (!interface.internal) {
      console.log(`- ${interfaceName}: ${interface.address} (${interface.family})`);
    }
  });
});
console.log('');

// Test port binding
const express = require('express');
const app = express();

app.get('/debug', (req, res) => {
  res.json({
    message: 'Debug endpoint working',
    port: process.env.PORT,
    host: req.get('host'),
    protocol: req.protocol,
    url: `${req.protocol}://${req.get('host')}`,
    headers: req.headers,
    timestamp: new Date().toISOString()
  });
});

const PORT = process.env.PORT || 10000;
console.log(`Attempting to bind to port: ${PORT}`);

app.listen(PORT, '0.0.0.0', () => {
  console.log('✅ Successfully bound to port:', PORT);
  console.log('✅ Server is listening on 0.0.0.0:' + PORT);
  console.log('');
  console.log('Test URLs:');
  console.log('- Local: http://localhost:' + PORT + '/debug');
  if (process.env.RENDER_EXTERNAL_URL) {
    console.log('- External:', process.env.RENDER_EXTERNAL_URL + '/debug');
  }
  console.log('');
  console.log('🎉 Debug server started successfully!');
}).on('error', (err) => {
  console.error('❌ Failed to bind to port:', PORT);
  console.error('Error:', err.message);
  process.exit(1);
});
