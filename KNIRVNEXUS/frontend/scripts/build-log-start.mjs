import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import os from 'os';

const logFilePath = path.join(process.cwd(), 'build.log');

try {
  const gitHash = execSync('git rev-parse --short HEAD 2>/dev/null', { encoding: 'utf8' }).trim() || 'unknown';
  const npmVersion = execSync('npm --version', { encoding: 'utf8' }).trim();

  const logData = {
    buildTimestamp: new Date().toISOString(),
    gitHash,
    buildStatus: 'building',
    nodeVersion: process.version,
    npmVersion,
    buildHost: os.hostname(),
  };

  fs.writeFileSync(logFilePath, JSON.stringify(logData, null, 2));
  console.log('Build log initialized.');
} catch (error) {
  console.error('Failed to create build log:', error);
  const minimalLog = { buildStatus: 'failed_to_log_start', error: error.message };
  fs.writeFileSync(logFilePath, JSON.stringify(minimalLog, null, 2));
}