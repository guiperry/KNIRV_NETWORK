# KNIRV GraphChain Explorer

A real-time KNIRV GraphChain data explorer integrated into the KNIRVGATEWAY platform. This application provides comprehensive monitoring and analysis of SkillNodes, ErrorNodes, and KNIRV GraphChain network statistics.

## Overview

KNIRV GraphChain Explorer is a vanilla JavaScript implementation that migrated from the original React-based KNIRVGRAPH frontend. It provides:

- **Real-time Dashboard**: Live KNIRV GraphChain statistics and recent activity
- **SkillNodes Browser**: Explore and filter SkillNodes with advanced search
- **ErrorNodes Monitor**: Track and analyze ErrorNodes and their resolution
- **Server-Sent Events**: Real-time updates via SSE integration
- **Responsive Design**: Mobile-first design with glass morphism styling

## Architecture

### Frontend Stack
- **Vanilla JavaScript**: ES6+ modules with component-based architecture
- **CSS3**: Custom CSS with KNIRVGATEWAY color scheme integration
- **HTML5**: Semantic markup with accessibility features
- **Server-Sent Events**: Real-time data streaming

### Backend Integration
- **Netlify Functions**: KNIRV GraphChain API proxy and SSE endpoints
- **KNIRVGATEWAY**: Integrated navigation and styling
- **KNIRV GraphChain API**: Direct integration with KNIRV GraphChain backend

## Directory Structure

```
graphchain-explorer/
├── index.html              # Dashboard page
├── skills.html             # SkillNodes browser
├── errors.html             # ErrorNodes monitor

├── test-data.js            # Mock data for development
├── README.md               # This file
├── css/
│   ├── graphchain.css      # Core styles with KNIRVGATEWAY theme
│   ├── components.css      # Component-specific styles
│   └── responsive.css      # Mobile-responsive styles
├── js/
│   ├── graphchain-core.js  # Core application framework
│   ├── graphchain-api.js   # API client with caching
│   ├── graphchain-sse.js   # SSE client implementation
│   ├── components/
│   │   ├── skill-card.js   # SkillNode card component
│   │   ├── error-card.js   # ErrorNode card component
│   │   └── stats-card.js   # Statistics card component
│   └── pages/
│       ├── dashboard.js    # Dashboard controller
│       ├── skills.js       # Skills page controller
│       └── errors.js       # Errors page controller
└── assets/
    ├── icons/              # Custom icons
    └── images/             # Images and graphics
```

## Features

### Dashboard
- **Live Statistics**: Real-time GraphChain height, node counts, and performance metrics
- **Recent Activity**: Latest SkillNodes with performance indicators
- **Connection Status**: Live SSE connection monitoring
- **Quick Actions**: Direct navigation to detailed views

### SkillNodes Browser
- **Advanced Search**: Full-text search across skill types and capabilities
- **Smart Filtering**: Filter by validation status and performance metrics
- **Detailed Cards**: Rich information display with performance data
- **Pagination**: Efficient loading of large datasets

### ErrorNodes Monitor
- **Error Tracking**: Comprehensive error monitoring and analysis
- **Severity Filtering**: Filter by error severity levels
- **Resolution Status**: Track error resolution progress
- **Related Skills**: View SkillNodes associated with each error

### Real-time Updates
- **Server-Sent Events**: Live data streaming from GraphChain
- **Auto-refresh**: Intelligent background data updates
- **Connection Recovery**: Automatic reconnection on network issues
- **Event Buffering**: Reliable event delivery

## API Integration

### GraphChain API Client
```javascript
// Get current GraphChain height
const height = await graphChainAPI.getHeight();

// Fetch all SkillNodes
const skills = await graphChainAPI.getSkills();

// Search nodes
const results = await graphChainAPI.searchNodes('query');

// Get comprehensive stats
const stats = await graphChainAPI.getGraphChainStats();
```

### SSE Event Handling
```javascript
// Listen for height updates
graphChainSSE.on('height_changed', (height) => {
    updateHeightDisplay(height);
});

// Handle new skills
graphChainSSE.on('skill_added', (skill) => {
    addSkillToDisplay(skill);
});

// Monitor connection status
graphChainSSE.on('connection:open', () => {
    showConnectedStatus();
});
```

## Component System

### SkillCard Component
```javascript
const skill = {
    id: 'skill_001',
    skill_type: 'Natural Language Processing',
    capabilities: ['text_analysis', 'sentiment_analysis'],
    validation: { is_validated: true, validation_score: 0.92 },
    performance: { success_rate: 0.89, avg_resolution_time: 2.3 }
};

const card = new SkillCard(skill, container);
const element = card.render();
container.appendChild(element);
```

### ErrorCard Component
```javascript
const error = {
    id: 'error_001',
    error_type: 'Memory Allocation Error',
    description: 'Insufficient memory for processing',
    severity: 3,
    resolution_status: 'pending'
};

const card = new ErrorCard(error, container);
const element = card.render();
container.appendChild(element);
```

## Styling

### Color Scheme
The application uses the KNIRVGATEWAY color palette:
- **Primary Blue**: `#2563eb`
- **Primary Purple**: `#7c3aed`
- **Success Green**: `#059669`
- **Warning Orange**: `#d97706`
- **Error Red**: `#dc2626`

### Glass Morphism
Modern glass morphism effects with:
- Backdrop blur filters
- Semi-transparent backgrounds
- Subtle border highlights
- Smooth hover transitions

### Responsive Design
- **Mobile-first**: Optimized for mobile devices
- **Breakpoints**: Tablet, desktop, and ultra-wide support
- **Flexible Grid**: CSS Grid and Flexbox layouts
- **Touch-friendly**: Large touch targets and gestures

## Development

### Quick Start
```bash
# Start KNIRVGATEWAY development server
npm run dev

# Access KNIRV GraphChain Explorer
open http://localhost:8888/graphchain-explorer/
```

### Mock Data
For development and testing, the application includes comprehensive mock data:
```javascript
// Enable mock mode (automatic on localhost)
window.graphChainAPI = new MockGraphChainAPI();
window.graphChainSSE = new MockGraphChainSSEClient();
```

### Testing
Run the comprehensive test suite:
```bash
# Start development server
npm run dev

# Run KNIRV GraphChain Explorer tests (from project root)
cd ../integration-tests
./config/run-tests.sh graphchain-explorer

# Or run all JavaScript tests
./config/run-tests.sh javascript
```

### Build Process
No build process required - pure vanilla JavaScript and CSS.

## Deployment

### Netlify Configuration
The application is configured for Netlify deployment with:
- **Static hosting**: Direct file serving
- **Function proxying**: API and SSE endpoints
- **SPA routing**: Client-side navigation support

### Environment Variables
```bash
KNIRVGRAPH_URL=https://graph.knirv.com
TESTNET_MODE=false
NODE_ENV=production
```

## Performance

### Optimization Features
- **API Caching**: Intelligent request caching with TTL
- **Event Debouncing**: Optimized search and filter performance
- **Lazy Loading**: Progressive content loading
- **Memory Management**: Proper component cleanup

### Metrics
- **First Paint**: < 1.5s
- **Interactive**: < 2.5s
- **Bundle Size**: ~150KB (uncompressed)
- **Memory Usage**: < 50MB typical

## Browser Support

- **Chrome**: 90+
- **Firefox**: 88+
- **Safari**: 14+
- **Edge**: 90+

## Migration Notes

### From React to Vanilla JS
- **Component System**: Custom component architecture
- **State Management**: Event-driven state updates
- **Routing**: Simple hash-based routing
- **Event Handling**: Native DOM event listeners

### KNIRVGATEWAY Integration
- **Navigation**: Integrated header and menu system
- **Styling**: Consistent color scheme and typography
- **Assets**: Shared images and icons
- **Functions**: Netlify Functions for API proxying

## Contributing

1. Follow the existing code style and patterns
2. Add tests for new functionality
3. Update documentation for API changes
4. Test across supported browsers
5. Ensure mobile responsiveness

## License

Part of the KNIRV Network ecosystem. See main repository for license details.
