# Consolidated Documentation

# KNIRVANA TypeScript Client: The Experiential Gateway to the KNIRV D-TEN Ecosystem

<div align="center">

![TypeScript](https://img.shields.io/badge/TypeScript-5.6.3-blue?style=flat-square&logo=typescript)
![React](https://img.shields.io/badge/React-18.3.1-blue?style=flat-square&logo=react)
![Three.js](https://img.shields.io/badge/Three.js-0.170.0-green?style=flat-square&logo=three.js)
![Vite](https://img.shields.io/badge/Vite-5.4.19-purple?style=flat-square&logo=vite)

*High-performance web-based KNIRVANA client with immersive 3D graphics and real-time multiplayer*

</div>

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
    - [Frontend Stack](#frontend-stack)
    - [Backend Stack](#backend-stack)
    - [3D Graphics Pipeline](#3d-graphics-pipeline)
    - [Game Logic and State Management](#game-logic-and-state-management)
    - [User Interface and Experience](#user-interface-and-experience)
- [Quick Start](#quick-start)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation)
    - [Development Commands](#development-commands)
- [Game Controls](#game-controls)
    - [RTS-Style Camera Controls](#rts-style-camera-controls)
    - [Mouse Controls](#mouse-controls)
- [Development Guide](#development-guide)
    - [Project Structure](#project-structure)
    - [Key Components](#key-components)
        - [Game Components](#game-components)
        - [UI Components](#ui-components)
    - [State Management](#state-management)
    - [3D Scene Development](#3d-scene-development)
        - [Creating Game Objects](#creating-game-objects)
        - [Custom Shaders](#custom-shaders)
    - [API Integration](#api-integration)
        - [Backend Routes](#backend-routes)
        - [Frontend API Calls](#frontend-api-calls)
- [Styling & Theming](#styling--theming)
    - [TRON Aesthetic Implementation](#tron-aesthetic-implementation)
    - [Responsive Design](#responsive-design)
- [KNIRV Ecosystem Integration](#knirv-ecosystem-integration)
    - [Blockchain Connectivity](#blockchain-connectivity)
    - [Real-Time Synchronization](#real-time-synchronization)
- [Testing](#testing)
    - [Running Tests](#running-tests)
    - [Testing Strategy](#testing-strategy)
- [Deployment](#deployment)
    - [Production Build](#production-build)
    - [Environment Configuration](#environment-configuration)
    - [Docker Deployment](#docker-deployment)
- [Contributing](#contributing)
    - [Code Style](#code-style)
- [License](#license)


## Overview

KNIRVANA is a high-performance, cross-platform Real-Time Strategy (RTS) game serving as the experiential gateway to the KNIRV Decentralized Trusted Execution Network (D-TEN).  This full-stack web application, built with modern React and Three.js, provides an immersive 3D TRON-style gaming experience where players command AI agents to competitively resolve ErrorNodes within the living KNIRVGRAPH.  Players earn NRN tokens through collective intelligence and participate in a self-healing decentralized network.


## Features

- **🎮 Immersive 3D Gameplay**: WebGL-powered graphics with Three.js and React Three Fiber
- **⚡ Real-Time Strategy**: RTS-style camera controls and competitive multiplayer
- **🌐 Cross-Platform**: Runs on desktop and mobile browsers
- **🔗 Blockchain Integration**: Native NRN token support and XION blockchain connectivity
- **🎨 TRON Aesthetic**: Distinctive neon grid environments with particle effects
- **📱 Responsive Design**: Adaptive UI for different screen sizes and devices


## Architecture

### Frontend Stack
- **React 18** - Component-based UI framework with modern hooks
- **TypeScript** - Static type checking and enhanced developer experience
- **React Three Fiber** - Declarative 3D scene management with Three.js
- **Vite** - Fast build tool with hot module replacement
- **Tailwind CSS** - Utility-first styling framework
- **Radix UI** - Accessible component primitives
- **shadcn/ui** - Pre-built component library based on Radix UI primitives
- **Lucide React** - Icon library


### Backend Stack
- **Express.js** - Node.js web framework for REST API
- **Drizzle ORM** - Type-safe database toolkit
- **PostgreSQL** - Primary database via Neon serverless
- **WebSocket** - Real-time communication for multiplayer features


### 3D Graphics Pipeline
- **Three.js** - WebGL-based 3D graphics library
- **React Three Fiber** - React renderer for Three.js scenes
- **@react-three/drei** - Useful helpers and abstractions
- **@react-three/postprocessing** - Post-processing effects
- **GLSL Shaders** - Custom shader support via vite-plugin-glsl


### Game Logic and State Management
The game state is managed through Zustand stores, providing reactive state updates across React components and 3D entities. Game mechanics include agent deployment, node resolution, resource management, and real-time competitive gameplay.  Key design decisions include reactive game state, modular game systems, and real-time interaction.


### User Interface and Experience
The UI combines game HUD elements with traditional web interfaces, using Tailwind CSS for styling and Radix UI for accessible components. The interface adapts between menu states and active gameplay, providing contextual information and controls. Key design decisions include immersive HUD design, progressive disclosure, and cross-device compatibility.


## Quick Start

### Prerequisites
- Node.js 18+
- npm or yarn package manager
- Modern web browser with WebGL support

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd KNIRVANA/ts-client

# Install dependencies
npm install

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# Start development server
npm run dev
```

### Development Commands

```bash
# Development server with hot reload
npm run dev

# Type checking
npm run check

# Build for production
npm run build

# Start production server
npm start

# Database operations
npm run db:push
```

## Game Controls

### RTS-Style Camera Controls
- **WASD** - Camera movement (forward/backward/left/right)
- **Q/E** - Camera rotation (left/right)
- **+/-** - Zoom in/out
- **Space** - Select objects
- **R** - Deploy agents

### Mouse Controls
- **Click** - Select ErrorNodes and agents
- **Drag** - Camera panning
- **Scroll** - Zoom control
- **Right-click** - Context actions


## Development Guide

### Project Structure

```
ts-client/
├── client/                 # Frontend React application
│   ├── src/
│   │   ├── components/     # React components
│   │   │   ├── game/      # Game-specific components
│   │   │   └── ui/        # UI components
│   │   ├── lib/           # Utilities and stores
│   │   ├── hooks/         # Custom React hooks
│   │   ├── types/         # TypeScript type definitions
│   │   └── shaders/       # GLSL shader files
│   ├── public/            # Static assets
│   │   ├── textures/      # 3D textures
│   │   ├── geometries/    # 3D models
│   │   └── sounds/        # Audio files
│   └── index.html         # Entry HTML file
├── server/                # Backend Express.js application
│   ├── index.ts          # Server entry point
│   ├── routes.ts         # API routes
│   ├── storage.ts         # Database abstraction
│   └── vite.ts           # Vite middleware
├── shared/               # Shared types and schemas
│   └── schema.ts         # Database schema
└── package.json          # Dependencies and scripts
```

### Key Components

#### Game Components
- **KnirvanaGame** - Main game canvas and scene setup
- **GameScene** - 3D scene management and rendering
- **GameUI** - HUD and interface overlays
- **KnirvGraph** - KNIRVGRAPH visualization
- **CameraController** - RTS-style camera controls

#### UI Components
- **GameHUD** - Real-time game information display
- **NodeInfo** - ErrorNode details and interaction
- **AgentPanel** - Agent management interface
- **ResourceDisplay** - NRN balance and statistics

### State Management

The application uses Zustand for state management.  Example:

```typescript
// Game state store
const useKnirvana = create((set, get) => ({
  gamePhase: 'menu',
  selectedErrorNode: null,
  selectedAgent: null,
  errorNodes: [],
  agents: [],
  playerResources: { nrn: 1000, skills: [] },
  // ... actions
}));

// Audio state store
const useAudio = create((set) => ({
  volume: 0.5,
  muted: false,
  // ... actions
}));
```

### 3D Scene Development

#### Creating Game Objects

```typescript
// Example ErrorNode component
function ErrorNode({ position, type, difficulty }) {
  const meshRef = useRef();
  
  useFrame((state, delta) => {
    // Animation logic
    meshRef.current.rotation.y += delta;
  });

  return (
    <mesh ref={meshRef} position={position}>
      <boxGeometry args={[1, 1, 1]} />
      <meshStandardMaterial 
        color={getColorByType(type)}
        emissive={0x444444}
      />
    </mesh>
  );
}
```

#### Custom Shaders

```glsl
// Example vertex shader (shaders/grid.vert)
varying vec2 vUv;

void main() {
  vUv = uv;
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
```

### API Integration

#### Backend Routes

```typescript
// Example API route
app.get('/api/game/state', async (req, res) => {
  const gameState = await storage.getGameState(req.user.id);
  res.json(gameState);
});

app.post('/api/agents/deploy', async (req, res) => {
  const { agentId, targetNodeId } = req.body;
  const result = await deployAgent(agentId, targetNodeId);
  res.json(result);
});
```

#### Frontend API Calls

```typescript
// Using TanStack Query for API state management
const { data: gameState } = useQuery({
  queryKey: ['gameState'],
  queryFn: () => fetch('/api/game/state').then(res => res.json()),
  refetchInterval: 1000, // Real-time updates
});
```

## Styling & Theming

### TRON Aesthetic Implementation

The game uses a distinctive TRON-style visual design:

```css
/* Example TRON-style components */
.error-node {
  @apply border border-cyan-400 bg-cyan-400/10 shadow-lg shadow-cyan-400/50;
  backdrop-filter: blur(8px);
}

.neon-text {
  color: #00ffff;
  text-shadow: 0 0 10px #00ffff, 0 0 20px #00ffff, 0 0 30px #00ffff;
}

.grid-background {
  background-image: 
    linear-gradient(cyan 1px, transparent 1px),
    linear-gradient(90deg, cyan 1px, transparent 1px);
  background-size: 50px 50px;
  opacity: 0.3;
}
```

### Responsive Design

```typescript
// Adaptive UI based on screen size
const isMobile = useMediaQuery('(max-width: 768px)');

return (
  <div className={cn(
    "game-ui",
    isMobile ? "mobile-layout" : "desktop-layout"
  )}>
    {/* UI components */}
  </div>
);
```

## KNIRV Ecosystem Integration

### Blockchain Connectivity

```typescript
// NRN token integration
const burnNRN = async (amount: number, skillId: string) => {
  const transaction = await knirvSDK.economics.burn({
    amount,
    metadata: { skillId, gameSession: sessionId }
  });
  return transaction;
};
```

### Real-Time Synchronization

```typescript
// WebSocket connection for multiplayer
const ws = new WebSocket('wss://api.knirv.com/game');

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  updateGameState(update);
};
```

## Testing

### Running Tests

```bash
# Unit tests
npm run test

# E2E tests
npm run test:e2e

# Visual regression tests
npm run test:visual
```

### Testing Strategy

- **Unit Tests**: Component logic and utility functions
- **Integration Tests**: API endpoints and database operations
- **E2E Tests**: Complete user workflows
- **Performance Tests**: 3D rendering and game loop optimization


## Deployment

### Production Build

```bash
# Build both client and server
npm run build

# Start production server
npm start
```

### Environment Configuration

```env
# .env file
NODE_ENV=production
DATABASE_URL=postgresql://...
KNIRV_API_ENDPOINT=https://api.knirv.com
XION_CHAIN_ID=knirv-mainnet-1
```

### Docker Deployment

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY dist ./dist
EXPOSE 3000
CMD ["npm", "start"]
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes following the code style
4. Add tests for new functionality
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to the branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Code Style

- Use TypeScript for all new code
- Follow React best practices and hooks patterns
- Use Tailwind CSS for styling
- Write comprehensive tests
- Document complex 3D graphics code

## License

This project is part of the KNIRV Network ecosystem.

---

<div align="center">

**Ready to command AI agents in the KNIRVGRAPH?**

[🎮 Start Playing](https://knirvana.knirv.network) • [📚 Documentation](https://docs.knirv.network) • [💬 Community](https://discord.gg/knirv)

</div>


## ECOSYSTEM GAME INTEGRATION

# Ecosystem Game Integration - Implementation Summary

## ✅ Integration Complete

The KNIRVANA Ecosystem Game has been successfully integrated into the ts-client as a slide-out overlay panel with floating boxes over the 3D graph animation background.

## 🎯 What Was Accomplished

### 1. **Seamless UI Integration**
- Added "🎮 Ecosystem Game" button to the top HUD in GameUI.tsx
- Created slide-out panel that appears from the right side
- Maintained 3D graph animation in the background with semi-transparent overlay
- Implemented smooth Framer Motion animations for panel transitions

### 2. **Complete Game Functionality**
- **767 lines** of fully functional ecosystem game code
- All original game mechanics preserved and optimized for overlay display
- Two game modes: Minimal (Step 1) and Full (Step 2)
- Complete skill system with 4 upgradeable bot skills
- Interactive 4x4 cluster grid with floating error boxes
- Real-time resource management with Knerv currency

### 3. **Advanced Game Features**
- **Bot Control Panel**: Debugging, ML, Security, Optimization skills
- **Ecosystem Map**: Visual cluster grid with severity indicators
- **Error Fixing System**: Complex skill requirements and cascading errors
- **Avatar Builder**: Attribute point allocation (Full mode)
- **Leaderboard**: Competitive scoring with AI opponents
- **Progress Charts**: Performance tracking with Recharts integration

### 4. **Technical Excellence**
- **Zero Dependency Conflicts**: All required packages already installed
- **TypeScript Safety**: Complete type definitions and interfaces
- **Responsive Design**: Adapts to different screen sizes
- **Performance Optimized**: Lazy-loaded components and efficient state management

## 🛠 Files Created/Modified

### New Files:
- `KNIRVANA/ts-client/client/src/components/ui/EcosystemGameSlideout.tsx` (767 lines)
- `KNIRVANA/INTEGRATION_GUIDE.md` (comprehensive documentation)

### Modified Files:
- `KNIRVANA/ts-client/client/src/components/ui/GameUI.tsx` (added button and slideout integration)

## 🎮 User Experience Flow

1. **Entry Point**: User clicks purple "🎮 Ecosystem Game" button in top-right HUD
2. **Smooth Transition**: Panel slides in from right with backdrop overlay
3. **Immersive Gaming**: Full ecosystem game functionality with floating panels
4. **Background Context**: 3D graph animation continues behind semi-transparent backdrop
5. **Easy Exit**: Close via X button or backdrop click for seamless return

## 🔧 Technical Architecture

### Component Structure:
```
GameUI (Modified)
├── Ecosystem Game Button (New)
└── EcosystemGameSlideout (New)
    ├── Header with Mode Toggle
    ├── Game Content Grid
    │   ├── Bot Control Panel
    │   ├── Ecosystem Map/Cluster View
    │   └── Leaderboard/Progress Charts
    └── All Game Logic & State Management
```

### State Management:
- **Independent Game State**: Completely isolated from main app state
- **Local State Management**: Uses React hooks for game progression
- **Persistent UI State**: Slideout open/close state managed in GameUI

## 🚀 Running the Integration

The integration is now live and running:
- **Server**: Running on port 5000 (`npm run dev`)
- **Access**: Click "🎮 Ecosystem Game" button in the KNIRVANA interface
- **Testing**: All game features functional and responsive

## 🎯 Key Benefits Achieved

### 1. **Unified Experience**
- Single application with dual functionality
- Seamless context switching between 3D visualization and game
- Consistent design language and user interface

### 2. **Enhanced Engagement**
- Interactive learning through gameplay
- Skill development in debugging and system optimization
- Competitive elements with leaderboards and progression

### 3. **Technical Robustness**
- Clean, maintainable code architecture
- Full TypeScript type safety
- Optimized performance with no conflicts

### 4. **Future-Ready Design**
- Modular component structure for easy extensions
- Scalable state management approach
- Ready for additional game modes and features

## 🎉 Integration Success Metrics

- ✅ **Zero Build Errors**: Clean compilation and runtime
- ✅ **All Dependencies Satisfied**: No additional packages needed
- ✅ **Complete Functionality**: All original game features working
- ✅ **Smooth Performance**: Responsive animations and interactions
- ✅ **Visual Consistency**: Matches KNIRVANA design system
- ✅ **User Experience**: Intuitive navigation and controls

## 🔮 Next Steps

The integration is complete and fully functional. The ecosystem game now operates as a seamless overlay within the KNIRVANA ts-client, providing users with an immersive debugging and skill management experience while maintaining the context of the 3D graph visualization.

Users can now:
- Access the full ecosystem game via the top HUD button
- Enjoy floating game panels over the animated background
- Switch between game modes and manage bot skills
- Track progress and compete on leaderboards
- Return seamlessly to the main 3D interface

The integration successfully transforms the standalone ecosystem game into a cohesive part of the KNIRVANA experience, creating a unified platform for both visualization and interactive learning.
