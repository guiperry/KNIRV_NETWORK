#!/usr/bin/env node

const childProcess = require('node:child_process');
const fs = require('node:fs');
const http = require('node:http');
const https = require('node:https');
const path = require('node:path');

const DEFAULT_BASE_URL = 'https://releases.knirv.com/engine/installer';
const packageRoot = path.resolve(__dirname, '..');

async function main(argv = process.argv.slice(2), environment = process.env) {
  const args = new Set(argv);
  const dryRun = args.has('--dry-run');
  const noStart = args.has('--no-start') || environment.KNIRVENGINE_INSTALLER_NO_START === '1';
  const platform = detectPlatform();
  const baseURL = trimTrailingSlash(environment.KNIRVENGINE_INSTALLER_BASE_URL || DEFAULT_BASE_URL);
  const installerName = installerFileName(platform);
  const installDir = path.resolve(environment.KNIRVENGINE_INSTALLER_DIR || path.join(packageRoot, 'bin', 'native', platform.key));
  const installerPath = path.join(installDir, installerName);
  const downloadURL = `${baseURL}/${installerName}`;

  log(`Detected platform: ${platform.key}`);
  log(`Downloading installer: ${downloadURL}`);
  log(`Installer path: ${installerPath}`);

  if (!dryRun) {
    fs.mkdirSync(installDir, { recursive: true });
    await downloadFile(downloadURL, installerPath);
    if (platform.os !== 'windows') {
      fs.chmodSync(installerPath, 0o755);
    }
  }

  if (noStart) {
    log('Skipping installer initialization.');
    return installerPath;
  }
  if (dryRun) {
    log('Dry run: installer would now initialize KNIRVENGINE.');
    return installerPath;
  }

  // Keep the npm process attached: the Go installer may need sudo, macOS
  // authentication, or Windows UAC. A detached child would hide that prompt.
  const result = childProcess.spawnSync(installerPath, [], {
    stdio: 'inherit',
    windowsHide: false,
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`KNIRVENGINE installer exited with status ${result.status}`);
  }
  log('KNIRVENGINE installer completed.');
  return installerPath;
}

function detectPlatform(platform = process.platform, arch = process.arch) {
  const osMap = { darwin: 'macos', linux: 'linux', win32: 'windows' };
  const archMap = { x64: 'amd64', arm64: 'arm64' };
  const os = osMap[platform];
  const cpu = archMap[arch];
  if (!os || !cpu || (os === 'windows' && cpu !== 'amd64')) {
    throw new Error(`Unsupported platform: ${platform}/${arch}`);
  }
  return { os, arch: cpu, key: `${os}_${cpu}` };
}

function installerFileName(platform) {
  return `knirvengine-installer-${platform.os}-${platform.arch}${platform.os === 'windows' ? '.exe' : ''}`;
}

async function downloadFile(url, destination) {
  const temporary = `${destination}.download-${process.pid}`;
  const file = fs.createWriteStream(temporary, { mode: 0o755 });
  try {
    await new Promise((resolve, reject) => {
      downloadToStream(url, file, 0, (error) => error ? reject(error) : file.end(resolve));
    });
    fs.renameSync(temporary, destination);
  } catch (error) {
    file.close(() => {});
    fs.rmSync(temporary, { force: true });
    throw error;
  }
}

function downloadToStream(url, file, redirects, callback) {
  const parsed = new URL(url);
  const client = parsed.protocol === 'http:' ? http : https;
  const request = client.get(parsed, (response) => {
    if ([301, 302, 303, 307, 308].includes(response.statusCode) && response.headers.location) {
      response.resume();
      if (redirects >= 5) return callback(new Error(`Too many redirects downloading ${url}`));
      return downloadToStream(new URL(response.headers.location, parsed).toString(), file, redirects + 1, callback);
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      response.resume();
      return callback(new Error(`Download failed: HTTP ${response.statusCode} ${url}`));
    }
    response.pipe(file, { end: false });
    response.on('end', () => callback());
  });
  request.on('error', callback);
}

function trimTrailingSlash(value) { return value.replace(/\/+$/, ''); }
function log(message) { console.log(`[knirvengine-install] ${message}`); }

module.exports = { DEFAULT_BASE_URL, detectPlatform, installerFileName, main, trimTrailingSlash };

if (require.main === module) {
  main().catch((error) => {
    console.error(`[knirvengine-install] ${error.message}`);
    process.exit(1);
  });
}
