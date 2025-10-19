// server/index.ts
import { startServer } from './server';
import { logger } from './utils'; // Assuming logger is exported from utils

logger.log("Backend server starting...");
startServer().catch(error => {
    logger.error(`Failed to start backend server: ${error}`);
    (process.exit as (code?: number) => never)(1);
});
