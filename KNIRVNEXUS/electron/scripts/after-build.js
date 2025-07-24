// electron/scripts/after-build.js
// Post-build script for additional processing

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

exports.default = async function afterBuild(context) {
  console.log('Running post-build processing...');

  const { outDir, electronPlatformName } = context;
  
  try {
    // Generate checksums for all built files
    await generateChecksums(outDir);
    
    // Create build info file
    await createBuildInfo(context);
    
    // Platform-specific post-processing
    switch (electronPlatformName) {
      case 'win32':
        await processWindowsBuild(context);
        break;
      case 'darwin':
        await processMacOSBuild(context);
        break;
      case 'linux':
        await processLinuxBuild(context);
        break;
    }
    
    console.log('Post-build processing completed successfully');
  } catch (error) {
    console.error('Post-build processing failed:', error);
    throw error;
  }
};

async function generateChecksums(outDir) {
  console.log('Generating checksums...');
  
  const files = fs.readdirSync(outDir);
  const checksums = {};
  
  for (const file of files) {
    const filePath = path.join(outDir, file);
    const stats = fs.statSync(filePath);
    
    if (stats.isFile() && !file.endsWith('.sha256')) {
      const hash = crypto.createHash('sha256');
      const data = fs.readFileSync(filePath);
      hash.update(data);
      const checksum = hash.digest('hex');
      
      checksums[file] = {
        sha256: checksum,
        size: stats.size,
        modified: stats.mtime.toISOString()
      };
      
      // Write individual checksum file
      fs.writeFileSync(
        path.join(outDir, `${file}.sha256`),
        `${checksum}  ${file}\n`
      );
    }
  }
  
  // Write combined checksums file
  fs.writeFileSync(
    path.join(outDir, 'checksums.json'),
    JSON.stringify(checksums, null, 2)
  );
  
  console.log(`Generated checksums for ${Object.keys(checksums).length} files`);
}

async function createBuildInfo(context) {
  console.log('Creating build info...');

  const { outDir, packager } = context;

  // Handle case where packager might not be available (e.g., in pack mode)
  let appInfo = null;
  if (packager && packager.appInfo) {
    appInfo = packager.appInfo;
  }

  const buildInfo = {
    appName: appInfo ? appInfo.productName : 'Agentic Engine',
    version: appInfo ? appInfo.version : '1.0.0',
    buildVersion: appInfo ? appInfo.buildVersion : '1.0.0',
    platform: process.platform,
    arch: process.arch,
    electronVersion: process.versions.electron,
    nodeVersion: process.versions.node,
    buildDate: new Date().toISOString(),
    buildEnvironment: process.env.NODE_ENV || 'production',
    gitCommit: process.env.GIT_COMMIT || 'unknown',
    gitBranch: process.env.GIT_BRANCH || 'unknown'
  };
  
  fs.writeFileSync(
    path.join(outDir, 'build-info.json'),
    JSON.stringify(buildInfo, null, 2)
  );
  
  console.log('Build info created');
}

async function processWindowsBuild(context) {
  console.log('Processing Windows build...');
  
  // Add Windows-specific post-processing here
  // For example: code signing verification, installer customization, etc.
  
  const { outDir } = context;
  
  // Check if installer was created
  const installerFiles = fs.readdirSync(outDir).filter(file => 
    file.endsWith('.exe') && file.includes('Setup')
  );
  
  if (installerFiles.length > 0) {
    console.log(`Windows installer created: ${installerFiles[0]}`);
  }
}

async function processMacOSBuild(context) {
  console.log('Processing macOS build...');
  
  // Add macOS-specific post-processing here
  // For example: DMG customization, notarization verification, etc.
  
  const { outDir } = context;
  
  // Check if DMG was created
  const dmgFiles = fs.readdirSync(outDir).filter(file => file.endsWith('.dmg'));
  
  if (dmgFiles.length > 0) {
    console.log(`macOS DMG created: ${dmgFiles[0]}`);
  }
}

async function processLinuxBuild(context) {
  console.log('Processing Linux build...');
  
  // Add Linux-specific post-processing here
  // For example: AppImage verification, package validation, etc.
  
  const { outDir } = context;
  
  // Check what Linux packages were created
  const linuxFiles = fs.readdirSync(outDir).filter(file => 
    file.endsWith('.AppImage') || file.endsWith('.deb') || 
    file.endsWith('.rpm') || file.endsWith('.tar.gz')
  );
  
  if (linuxFiles.length > 0) {
    console.log(`Linux packages created: ${linuxFiles.join(', ')}`);
  }
}
