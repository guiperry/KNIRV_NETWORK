# Overview

This is a full-stack web application built with React and Express that features a 3D interactive KNIRV Network ecosystem. The project combines a modern React frontend with Three.js for 3D graphics, a PostgreSQL database with Drizzle ORM, and an Express backend. The application presents an immersive 3D interface where users can explore different KNIRV network services through an interactive menu system with pulsating orbs, ring systems, and electric shock effects.

# User Preferences

Preferred communication style: Simple, everyday language.

# System Architecture

## Frontend Architecture
- **Framework**: React 18 with TypeScript and Vite for fast development
- **3D Graphics**: Three.js ecosystem (@react-three/fiber, @react-three/drei) for WebGL rendering
- **UI Components**: Radix UI primitives with custom styling via Tailwind CSS
- **State Management**: Zustand for lightweight global state (game phase, audio controls)
- **Animations**: React Spring for smooth 3D transitions and animations
- **Styling**: Tailwind CSS with custom CSS variables for theming

## Backend Architecture
- **Runtime**: Node.js with Express.js framework
- **Language**: TypeScript with ES modules
- **API Structure**: RESTful endpoints under `/api` prefix
- **Error Handling**: Centralized error middleware with status code management
- **Development**: Hot reload via Vite middleware in development mode

## Database Layer
- **ORM**: Drizzle with PostgreSQL dialect for type-safe database operations
- **Schema**: User management system with username/password authentication
- **Storage**: Dual storage pattern - MemStorage for development, PostgreSQL for production
- **Migrations**: Drizzle Kit for schema migrations and database management

## Build System
- **Bundler**: Vite for frontend, esbuild for backend production builds
- **Development**: Integrated dev server with HMR and error overlay
- **Production**: Static file serving with optimized builds
- **Assets**: Support for 3D models (GLTF/GLB), audio files, and GLSL shaders

## 3D Scene Architecture
- **Scene Management**: Canvas-based 3D rendering with orbit controls
- **Component Structure**: Modular 3D components (PulsatingOrb, RingSystem, ElectricShock)
- **Performance**: Optimized WebGL settings with selective anti-aliasing and power preference
- **Interaction**: Click-based service selection with modal presentations

# External Dependencies

## Core Frameworks
- **React Three Fiber**: 3D scene graph for React applications
- **Drei**: Helper library for common Three.js patterns and controls
- **Express.js**: Web application framework for Node.js backend

## Database & ORM
- **PostgreSQL**: Primary database (via @neondatabase/serverless for cloud deployment)
- **Drizzle ORM**: Type-safe SQL toolkit with schema validation
- **Zod**: Runtime type validation for API schemas

## UI & Styling
- **Radix UI**: Accessible component primitives (dialogs, buttons, forms)
- **Tailwind CSS**: Utility-first CSS framework
- **Class Variance Authority**: Component variant management
- **Lucide React**: Icon library for UI elements

## Development Tools
- **Vite**: Fast build tool with HMR support
- **TypeScript**: Static type checking across the stack
- **ESBuild**: Fast JavaScript bundler for production builds
- **GLSL Plugin**: Shader support for advanced 3D effects

## State & Data Management
- **Zustand**: Lightweight state management for React
- **TanStack Query**: Server state management and caching
- **React Hook Form**: Form handling with validation

## Audio & Animations
- **React Spring**: Physics-based animations for 3D scenes
- **Web Audio API**: Browser-native audio for sound effects and music
- **GLSL Shaders**: Custom graphics programming for visual effects