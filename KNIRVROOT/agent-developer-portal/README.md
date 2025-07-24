# KNIRVROOT Developer Portal

This is the Developer Portal for KNIRVROOT, a browser-based interface for developers to interact with the KNIRVROOT network.

## Overview

The Developer Portal is a Node.js service that serves a Next.js-based web application. It provides a user interface for developers to:

- View blockchain information
- Manage NFT capabilities
- Interact with DAOs
- Monitor network status
- Manage settlements

## Setup

1. Install dependencies:
   ```
   npm install
   ```

2. Copy the Next.js static files to the `static` directory:
   ```
   mkdir -p static
   cp -r ../altgui/out/* static/
   ```

3. Start the server:
   ```
   npm start
   ```

## Configuration

The server can be configured using environment variables:

- `HTTP_API_PORT`: The port to run the server on (default: 3000)
- `API_KEY`: API key for protected routes (default: 'default-api-key')
- `NODE_ENV`: Environment mode (development/production)
- `CHAIN_ID`: The chain ID to connect to

## API Endpoints

- `/health`: Health check endpoint
- `/info`: Server information
- `/api/protected/config`: Protected configuration endpoint
- `/api/blockchain/*`: Proxy to blockchain API

## Development

For development with auto-reload:

```
npm run dev
```