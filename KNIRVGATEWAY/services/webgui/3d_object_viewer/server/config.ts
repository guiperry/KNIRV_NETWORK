// config.ts - Configuration management

import * as dotenv from 'dotenv';
import { Config, RPCConfig } from './types';
import { logger } from './utils';

// Load .env file if present
dotenv.config();

// Main application configuration
export const mainConfig: Config = {
    RPCEndpoint: "",
    ServiceAddress: "",
    Port: ""
};

// RPC Configuration
export const rpc_config: RPCConfig = {
    Port: ""
};

export const backendPort = parseInt(process.env.BACKEND_PORT || '3001', 10);

// Load application configuration from environment variables or config file
export function loadAppConfig(): void {
    logger.log("Loading application configuration...");
    
    // Try to load from environment variables first
    mainConfig.RPCEndpoint = process.env.RPC_ENDPOINT || "";
    mainConfig.ServiceAddress = process.env.SERVICE_ADDRESS || "localhost";
    mainConfig.Port = process.env.PORT || "3001";
    
    logger.log(`Configuration loaded: RPCEndpoint=${mainConfig.RPCEndpoint}, ServiceAddress=${mainConfig.ServiceAddress}, Port=${mainConfig.Port}`);
}

// Load RPC configuration
export function loadRPC_Config(): void {
    logger.log("Loading RPC configuration...");
    
    // Set RPC port from environment or use default
    rpc_config.Port = process.env.RPC_PORT || "8545";
    
    logger.log(`RPC configuration loaded: Port=${rpc_config.Port}`);
}