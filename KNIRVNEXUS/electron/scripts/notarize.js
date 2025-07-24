// electron/scripts/notarize.js
// macOS notarization script for code signing

const { notarize } = require('@electron/notarize');

exports.default = async function notarizing(context) {
  const { electronPlatformName, appOutDir } = context;
  
  if (electronPlatformName !== 'darwin') {
    return;
  }

  const appName = context.packager.appInfo.productFilename;

  // Check if we have the required environment variables
  const appleId = process.env.APPLE_ID;
  const appleIdPassword = process.env.APPLE_ID_PASSWORD;
  const teamId = process.env.APPLE_TEAM_ID;

  if (!appleId || !appleIdPassword || !teamId) {
    console.warn('Skipping notarization: APPLE_ID, APPLE_ID_PASSWORD, or APPLE_TEAM_ID not set');
    return;
  }

  console.log(`Notarizing ${appName}...`);

  try {
    await notarize({
      appBundleId: 'com.agenticengine.desktop',
      appPath: `${appOutDir}/${appName}.app`,
      appleId: appleId,
      appleIdPassword: appleIdPassword,
      teamId: teamId,
    });

    console.log('Notarization completed successfully');
  } catch (error) {
    console.error('Notarization failed:', error);
    throw error;
  }
};
