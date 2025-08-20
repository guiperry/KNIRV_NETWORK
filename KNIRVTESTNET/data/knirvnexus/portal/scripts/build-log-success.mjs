// Import necessary modules
import fs from 'fs';
import path from 'path';

const logFilePath = path.join(process.cwd(), 'build.log');

try {
  if (fs.existsSync(logFilePath)) {
    const logContent = fs.readFileSync(logFilePath, 'utf8');
    const logData = JSON.parse(logContent);
    logData.buildStatus = 'success';
    logData.buildCompletedAt = new Date().toISOString();
    fs.writeFileSync(logFilePath, JSON.stringify(logData, null, 2));
    console.log('Build log updated to success.');
  } else {
    throw new Error('build.log not found. Cannot update status.');
  }
} catch (error) {
  console.error('Failed to update build log:', error);
}