# KNIRVCHAIN 3D Object Viewer

A TypeScript application for viewing and managing 3D objects with blockchain integration. This project uses Next.js for the frontend and a custom Node.js server for the backend.

## Features

- **3D Model Viewing**: View 3D models in GLB, GLTF, and USDZ formats
- **Blockchain Integration**: Each 3D model is registered as an asset on the blockchain
- **Transaction Tracking**: All operations are recorded as transactions
- **Block Explorer**: View the blockchain structure and history
- **Interactive UI**: Rotate, pan, and zoom 3D models
- **Asset Management**: Upload and manage 3D assets

## Supported 3D Model Formats

- **GLB**: Binary form of glTF (GL Transmission Format) - recommended for web use
- **GLTF**: JSON-based format for 3D models with external resources
- **USDZ**: Apple's format for AR/VR content
- **Markdown**: Text files with .md extension for documentation

## Project Structure

The project has been refactored to separate the frontend and backend code:

- **Frontend**: Next.js application in the `pages` directory
- **Backend**: Node.js server in the `server` directory
- **Components**: Reusable React components in the `components` directory

### Frontend Structure

- `pages/index.tsx`: Main application page with 3D object viewer
- `pages/viewport.tsx`: Standalone 3D viewport page
- `pages/api/*.ts`: API routes that proxy requests to the backend server
- `pages/api/view/[id].ts`: Dynamic route for viewing a specific 3D object
- `components/Viewport.tsx`: Reusable 3D viewport component

### Backend Structure

- `server/index.ts`: Entry point for the backend server
- `server/server.ts`: Main server setup and route handling
- `server/config.ts`: Configuration management
- `server/utils.ts`: Utility functions and classes
- `server/db.ts`: Database operations
- `server/assets.ts`: Asset management
- `server/apiHandlers.ts`: HTTP API request handlers
- `server/rpcService.ts`: RPC service implementation
- `server/routes/objects.ts`: Routes for handling object requests
- `server/types.ts`: Type definitions

## Getting Started

### Installation

1. Clone the repository
2. Install dependencies:

```bash
npm install
```

### Running the Application

To run the application in development mode:

```bash
npm run dev
```

This will start both the Next.js frontend and the Node.js backend server concurrently.

Open [http://localhost:3000](http://localhost:3000) with your browser to see the frontend.

### Adding 3D Models

You can add 3D models in two ways:

1. **Through the application**:

   - Open the application in your browser
   - Click on the "Upload Object" button in the sidebar
   - Select a 3D model file (GLB, GLTF, or USDZ)
   - Enter a name for the model
   - Click "Upload to Chain"

2. **Manually**:
   - Add your 3D model files to the `public/assets/models` directory
   - Restart the application to scan for new models

### Viewing 3D Models

1. Open the application in your browser
2. Select an object from the list in the sidebar
3. The 3D model will appear in the viewport
4. Use the mouse to interact with the model:
   - Left-click and drag to rotate
   - Right-click and drag to pan
   - Scroll to zoom
   - Click "Reset View" to reset the camera

## Building for Production

To build the application for production:

```bash
npm run build
```

To start the production server:

```bash
npm start
```

## API Endpoints

### Frontend API (Next.js)

- `GET /api/objects`: Get a list of all 3D objects
- `GET /api/view/[id]`: View a specific 3D object
- `GET /api/blocks`: Get a list of all blockchain blocks
- `GET /api/assets`: Get a list of all assets
- `GET /api/transactions`: Get a list of all transactions

### Backend API (Node.js)

- `GET /api/objects`: Get a list of all 3D objects
- `GET /api/objects/[id]`: Get details of a specific 3D object
- `GET /api/blocks`: Get a list of all blockchain blocks
- `GET /api/assets`: Get a list of all assets
- `GET /api/transactions`: Get a list of all transactions
- `POST /register-content`: Register new content
- `POST /is-viewer-allowed`: Check if a viewer is allowed to access content
- `POST /add-viewer`: Add a viewer to content
- `POST /get-content-metadata`: Get content metadata

## RPC Methods

- `Multiply`: Multiply two integers
- `Divide`: Divide two integers
- `GetAssetDetails`: Get details of an asset

## Configuration

The application can be configured using environment variables or a `.env` file:

- `RPC_ENDPOINT`: RPC endpoint URL
- `SERVICE_ADDRESS`: Service address (default: "localhost")
- `PORT`: HTTP server port (default: 3000)
- `RPC_PORT`: RPC server port (default: 8545)

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.
