# KNIRV Desktop Client GUI

[TOC]

## Overview

This document provides comprehensive information about the KNIRV Network Desktop Client's frontend application.  It features a role-based interface for managing AI agents, skills, capabilities, and network operations.  This document covers features, development, deployment, architecture, configuration, migration details, testing, and troubleshooting.

## Features

- **Role-Based Access Control**: Secure access management with 5 user roles (Root, Bootnode, Peer, Client, User)
- **Nested Navigation**: Hierarchical navigation structure with expandable sub-menus
- **AI Agent Management**: Comprehensive agent lifecycle management and workflow orchestration
- **Skills Marketplace**: Decentralized AI skills marketplace (Skills DEX) with NRN token integration
- **MCP Capabilities**: Model Context Protocol capabilities management
- **Network Monitoring**: Real-time network monitoring and analytics
- **Glass Morphism UI**: Modern dark theme with glass morphism effects
- **Chat System**: Multi-step AI task execution (ChatChain) and personal AI with memory (MyChatBrain)
- **Monitoring System**: Real-time network monitoring, system performance metrics, and network explorers
- **Model Management**: AI models, skills, and capabilities inventory management, API provider configuration, and decentralized voting system
- **Capabilities Management**: Model Context Protocol capabilities management, distinct from the Skills marketplace
- **Properties Management**: Integration with existing NFT management functionality
- **API Management**: TunnelRegistry management with authentication and usage analytics


## Architecture

### Component Structure

The application follows a hierarchical component structure:

- **Main Components**: Top-level navigation components (Chat, Monitor, Models, etc.)
- **Sub-Components**: Nested functionality within each main section
- **Common Components**: Shared UI components with glass morphism styling (GlassCard, DataTable, SearchBar)
- **Role-Based Access**: Comprehensive access control system

### Navigation Structure

```
/dashboard
/chat/* (ChatChain, MyChatBrain)
/monitor/* (NetworkMonitor, LocalAnalytics, NetworkExplorers)
/models/* (CodexBuilder, FallbackConfig, DAOVoting)
/agents/* (MyAgents, MyTargets, MyWorkflows)
/skills/* (SkillsDEX)
/capabilities/* (CapabilityStore, MCPManager, MCPServers)
/properties/* (NFT IP Vault)
/api/* (Personal API Endpoints)
/settings
```

## Development

To start the development server:

```bash
npm run dev
```

## Building for Production

To build the application for production:

```bash
npm run build
```

This will create a `dist` directory with the compiled static files. These files are automatically served by the Go backend when running in production mode.

## Production Deployment

To run the entire application in production mode:

1. Build the frontend:
   ```bash
   cd gui
   npm run build
   cd ..
   ```

2. Run the Go server with the production flag:
   ```bash
   go run main.go --production
   ```

Alternatively, use the provided script:
```bash
./build_and_run.sh
```

## Configuration

The build configuration is defined in `vite.config.ts`. The application is configured to output static files to the `dist` directory, which is then served by the Go backend.

## Migration

The KNIRV Desktop Client GUI has undergone a comprehensive migration from altgui, implementing nested navigation with role-based access control.  This migration is complete.  Key changes include a hierarchical navigation structure, role-based access control for all pages and sub-pages, and a new component architecture using TypeScript and Tailwind CSS.  The new file structure is as follows:

```
KNIRVENGINE/desktop-client/gui/
├── src/
│   ├── components/
│   │   ├── common/             
│   │   │   ├── GlassCard.tsx   
│   │   │   ├── DataTable.tsx   
│   │   │   ├── SearchBar.tsx   
│   │   │   └── index.ts        
│   │   ├── chat/               
│   │   │   ├── ChatChain.tsx   
│   │   │   └── MyChatBrain.tsx 
│   │   ├── monitor/            
│   │   │   ├── NetworkMonitor.tsx    
│   │   │   ├── LocalAnalytics.tsx    
│   │   │   └── NetworkExplorers.tsx  
│   │   ├── models/             
│   │   │   ├── CodexBuilder.tsx      
│   │   │   ├── FallbackConfig.tsx    
│   │   │   └── DAOVoting.tsx         
│   │   ├── skills/             
│   │   │   └── SkillsDEX.tsx         
│   │   ├── AuthContext.tsx     
│   │   ├── Sidebar.tsx         
│   │   ├── Chat.tsx            
│   │   ├── Monitor.tsx         
│   │   ├── Models.tsx          
│   │   ├── Agents.tsx          
│   │   ├── Skills.tsx          
│   │   ├── Capabilities.tsx    
│   │   ├── Properties.tsx      
│   │   ├── API.tsx             
│   │   ├── Analytics.tsx       
│   │   └── Settings.tsx        
│   ├── styles/
│   │   └── knirv-theme.css     
│   └── tests/
│       └── access-control.test.ts  
├── README.md                   
```

## Testing

Run the test suite:

```bash
npm test
```

Run access control tests:

```bash
npm run test:access-control
```

## Role-Based Access Matrix

| Feature             | Root | Bootnode | Peer | Client | User |
|----------------------|------|----------|------|--------|------|
| Dashboard            | ✅   | ✅       | ✅   | ✅     | ✅   |
| Chat (ChatChain)     | ✅   | ✅       | ✅   | ❌     | ❌   |
| Chat (MyChatBrain)   | ✅   | ✅       | ✅   | ✅     | ✅   |
| Monitor (Network)    | ✅   | ✅       | ❌   | ❌     | ❌   |
| Monitor (Local)     | ✅   | ✅       | ✅   | ✅     | ✅   |
| Monitor (Explorers)  | ✅   | ✅       | ✅   | ❌     | ❌   |
| Models (All)         | ✅   | ✅       | Partial | Limited | Limited |
| Agents (All)         | ✅   | ✅       | ✅   | Partial | Partial |
| Skills DEX           | ✅   | ✅       | ✅   | ✅     | ✅   |
| Capabilities (Store) | ✅   | ✅       | ✅   | ✅     | ✅   |
| Capabilities (MCP)   | ✅   | ✅       | ✅   | ✅     | ❌   |
| Properties           | ✅   | ✅       | ✅   | ✅     | ✅   |
| API                  | ✅   | ✅       | ✅   | ✅     | ✅   |
| Settings             | ✅   | ✅       | ✅   | ✅     | ✅   |


## Troubleshooting

* **Navigation not working:** Ensure all routes use wildcard patterns (`/*`).
* **Access denied errors:** Check user role and access control matrices.
* **Styling issues:** Verify KNIRV theme CSS is properly imported.
* **Component not found:** Check component imports and file paths.


