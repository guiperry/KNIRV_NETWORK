# KNIRVANA - The Experiential Gateway

## Overview

KNIRVANA is a high-performance, cross-platform Real-Time Strategy (RTS) game that serves as the experiential gateway to the KNIRV Decentralized Trusted Execution Network (D-TEN). The application is a 3D TRON-style game where players command AI agents to competitively resolve ErrorNodes within the living KNIRVGRAPH. Built as a full-stack web application, it combines modern React with Three.js for immersive 3D graphics and Express.js for backend services.

The game transforms complex AI network operations into an engaging strategy experience, allowing players to deploy and manage autonomous AI agents, earn NRN tokens through collective intelligence, and participate in a self-healing decentralized network. Players compete in real-time to find and resolve ErrorNodes, with successful resolutions contributing to the broader KNIRV ecosystem's knowledge base.

## User Preferences

Preferred communication style: Simple, everyday language.

## System Architecture

### Frontend Architecture
The client-side application is built with React 18 and TypeScript, utilizing Vite as the build tool for optimal development experience and performance. The 3D game environment is powered by React Three Fiber (R3F) and Three.js, providing hardware-accelerated WebGL rendering. The UI layer combines shadcn/ui components with Radix UI primitives for consistent, accessible interface elements.

**Key Design Decisions:**
- **React Three Fiber Integration**: Chosen for seamless React integration with Three.js, enabling declarative 3D scene composition and efficient state management between game logic and 3D rendering
- **Component-Based 3D Architecture**: Game entities (ErrorNodes, AIAgents, SkillNodes) are implemented as reusable React components, promoting maintainability and consistent rendering patterns
- **Real-Time Strategy Controls**: Keyboard-based camera controls (WASD movement, Q/E rotation, +/- zoom) provide familiar RTS-style navigation
- **State Management**: Zustand stores handle game state, audio controls, and player interactions with minimal boilerplate

### Backend Architecture
The server uses Express.js with TypeScript in ES module format, providing a REST API foundation. The architecture follows a modular pattern with separate route handling, storage abstraction, and development-optimized Vite integration.

**Key Design Decisions:**
- **Storage Interface Pattern**: Implemented an abstract storage interface (`IStorage`) with an in-memory implementation, allowing easy migration to database solutions without changing business logic
- **Development-First Approach**: Vite middleware integration provides hot module replacement and optimized development experience
- **Express Middleware Stack**: Request logging, JSON parsing, and error handling are centrally managed through Express middleware

### 3D Rendering and Game Engine
The game engine leverages React Three Fiber for declarative 3D scene management, with custom components for each game entity type. The rendering pipeline includes dynamic lighting, particle systems, and post-processing effects to achieve the TRON-style aesthetic.

**Key Design Decisions:**
- **Declarative 3D Scene Graph**: Each game element (nodes, agents, environment) is a React component, enabling easy composition and state-driven updates
- **Performance-Optimized Rendering**: Uses Three.js object pooling patterns and React's reconciliation for efficient updates
- **Cross-Platform Compatibility**: Built for web deployment with mobile-responsive controls and adaptive UI elements

### Game Logic and State Management
The game state is managed through Zustand stores, providing reactive state updates across React components and 3D entities. Game mechanics include agent deployment, node resolution, resource management, and real-time competitive gameplay.

**Key Design Decisions:**
- **Reactive Game State**: Zustand enables automatic UI and 3D scene updates when game state changes
- **Modular Game Systems**: Separate stores for game logic, audio, and UI state promote clean separation of concerns
- **Real-Time Interaction**: Direct manipulation of 3D objects through React event handlers and Three.js raycasting

### User Interface and Experience
The UI combines game HUD elements with traditional web interfaces, using Tailwind CSS for styling and Radix UI for accessible components. The interface adapts between menu states and active gameplay, providing contextual information and controls.

**Key Design Decisions:**
- **Immersive HUD Design**: Game interface elements overlay the 3D scene with TRON-style aesthetics matching the game world
- **Progressive Disclosure**: Complex game information is revealed contextually based on player selection and game phase
- **Cross-Device Compatibility**: Responsive design supports desktop and mobile interfaces with appropriate control schemes

## External Dependencies

### Core Framework Dependencies
- **React 18**: Frontend framework providing component-based architecture and modern hooks
- **React Three Fiber**: React renderer for Three.js, enabling declarative 3D scene management
- **Three.js**: WebGL-based 3D graphics library for rendering game world and entities
- **Express.js**: Node.js web framework for REST API and static file serving
- **Vite**: Build tool and development server with hot module replacement

### Database and Storage
- **Drizzle ORM**: Type-safe database toolkit configured for PostgreSQL
- **Neon Database**: Cloud PostgreSQL provider (via @neondatabase/serverless)
- **PostgreSQL**: Primary database system for persistent data storage

### UI Component Libraries
- **Radix UI**: Headless UI primitives for accessible component foundation
- **shadcn/ui**: Pre-built component library based on Radix UI primitives
- **Tailwind CSS**: Utility-first CSS framework for styling and responsive design
- **Lucide React**: Icon library providing consistent iconography

### State Management and Data Fetching
- **Zustand**: Lightweight state management for game state and UI state
- **TanStack Query**: Server state management and caching for API interactions
- **React Hook Form**: Form state management and validation

### Development and Build Tools
- **TypeScript**: Static type checking for both frontend and backend code
- **ESLint**: Code linting and formatting consistency
- **PostCSS**: CSS processing with Tailwind CSS integration
- **tsx**: TypeScript execution for Node.js development server

### Game-Specific Libraries
- **@react-three/drei**: Three.js helpers and abstractions for common 3D patterns
- **@react-three/postprocessing**: Post-processing effects for enhanced visual quality
- **drei KeyboardControls**: Input handling for RTS-style camera controls