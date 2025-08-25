# KNIRV Chain Portal

GraphChain Explorer for the KNIRV Network - migrated from KNIRVGRAPH with updated terminology.

## Features

- **Vector Explorer**: Browse and analyze network vectors (formerly blocks)
- **Skill Registry**: Explore available skills and LoRA adapters
- **Error Node Tracking**: Monitor and analyze error patterns
- **Network Statistics**: Real-time network density and activity metrics
- **Graph Visualization**: Interactive visualization of the KNIRV GraphChain

## Terminology Updates

This portal uses updated KNIRV terminology:
- **Vectors** (formerly "blocks") - Core data structures in the GraphChain
- **Density** (formerly "height") - Network depth and activity measurement
- **Skills** - LoRA adapters containing weights and biases
- **Error Nodes** - Network error tracking and resolution points

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Run tests
npm test
```

## Architecture

Built with:
- React 18 + TypeScript
- Vite for fast development
- Tailwind CSS for styling
- React Router for navigation
- Lucide React for icons

## Integration

This portal integrates with:
- KNIRVGRAPH backend for data
- KNIRVCHAIN for vector information
- KNIRVORACLE for network statistics
- KNIRVROUTER for skill invocation data
