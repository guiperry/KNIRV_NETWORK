# KNIRVANA Ecosystem Game

A React-based interactive debugging and skill management game extracted from the KNIRVANA_P3.md specification.

## Overview

This single-file React component provides two modes:

- **Step 1: Minimal prototype** - Hub → Cluster → Simple Fix flow
- **Step 2: Full skeleton** - Skills, cascades, avatar points, leaderboard, charts

Use the mode toggle in the header to switch between Step 1 and Step 2.

## Features

### Core Gameplay
- **Bot Control & Skills**: Manage your debugging bot with upgradeable skills
- **Ecosystem Map**: Navigate through clusters of errors in a 4x4 grid
- **Error Fixing**: Use your bot's skills to resolve errors with varying complexity
- **Resource Management**: Earn and spend Knerv currency to upgrade skills

### Skills System
- **Debugging**: Syntax repair, trace analysis, unit patching
- **Machine Learning**: Pattern detection, anomaly prediction  
- **Security**: Exploit patching, encryption, access control
- **Optimization**: Runtime tuning, memory & IO efficiency

### Full Mode Features
- **Avatar Builder**: Allocate attribute points (Strength, Logic, Agility, Charisma)
- **Cascading Errors**: Fixes may trigger new errors in complex scenarios
- **Progress Charts**: Track your performance over time
- **Enhanced Leaderboard**: Compete with other bots

## Technology Stack

- **React 18** with TypeScript
- **Framer Motion** for smooth animations
- **Lucide React** for consistent iconography
- **Recharts** for data visualization
- **TailwindCSS** for styling
- **Vite** for fast development and building

## Getting Started

### Prerequisites
- Node.js 16+ 
- npm or yarn

### Installation

1. Navigate to the project directory:
```bash
cd KNIRVANA/ecosystem-game
```

2. Install dependencies:
```bash
npm install
```

3. Start the development server:
```bash
npm run dev
```

4. Open your browser to `http://localhost:3000`

### Build for Production

```bash
npm run build
```

The built files will be in the `dist` directory.

## Game Mechanics

### Basic Flow
1. Start with a debugging bot (Level 1 Debugging skill)
2. Click on clusters in the ecosystem map to view errors
3. Fix errors using your bot's skills (must meet complexity requirements)
4. Earn Knerv currency for successful fixes
5. Upgrade skills to handle more complex errors
6. Complete clusters to earn attribute points (Full mode)

### Skill Progression
- Skills start at level 0 (except Debugging at level 1)
- Cost increases exponentially: `baseCost * 1.6^currentLevel`
- Higher skill levels allow fixing more complex errors

### Error Complexity
- Errors require specific skills at minimum levels
- Complexity ranges from 1-3 (in full mode)
- Some errors require multiple skills simultaneously

### Scoring
- Base score: 100 points per resolved cluster
- Bonus: 25 points per severity level of resolved clusters
- Leaderboard tracks your progress against AI opponents

## Project Structure

```
ecosystem-game/
├── src/
│   ├── App.tsx          # Main game component
│   ├── main.tsx         # React entry point
│   └── index.css        # Global styles
├── index.html           # HTML template
├── package.json         # Dependencies and scripts
├── vite.config.ts       # Vite configuration
├── tsconfig.json        # TypeScript configuration
├── tailwind.config.js   # TailwindCSS configuration
└── postcss.config.js    # PostCSS configuration
```

## Development

The game is implemented as a single React component with multiple sub-components:

- `UntitledEcosystemGame` - Main game logic and state management
- `Header` - Mode switching and branding
- `BotPanel` - Skill management and currency display
- `AvatarPanel` - Attribute point allocation (full mode)
- `EcosystemMap` - Cluster grid visualization
- `ClusterView` - Error details and fixing interface
- `LeaderboardPanel` - Score tracking
- `ProgressChart` - Performance visualization (full mode)

## License

Part of the KNIRV Network ecosystem.
