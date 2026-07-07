# KNIRV WebGUI

A modern React-based web interface for the KNIRV network, providing comprehensive access to model management, network monitoring, and marketplace functionality.

## 🚀 Features

### Core Functionality
- **Dashboard**: Real-time network status and quick actions
- **Model Management**: Build, train, and deploy AI models
- **Network Monitoring**: Comprehensive network health and analytics
- **Marketplace**: Trade skills, capabilities, and properties
- **Personal Vault**: Manage models, wallets, skills, and assets
- **Multi-Network Support**: Connect to mainnet, testnet, or local networks

### Network Explorers
- **Graph Explorer**: Explore GraphChain network topology (embedded from `/graphchain-explorer`)
- **Chain Explorer**: Blockchain data and transaction explorer (embedded from `/knirvchain-portal`)
- **Oracle Explorer**: Oracle network monitoring and management

### User Experience
- **Hierarchical Navigation**: Expandable/collapsible menu structure
- **Role-Based Access**: Different views for different user roles
- **Responsive Design**: Mobile-friendly interface
- **Glass Morphism UI**: Modern, translucent design elements
- **Real-time Updates**: Live data feeds and notifications

## 🏗️ Architecture

### Frontend Structure
```
src/
├── components/          # Reusable UI components
│   ├── Footer.js       # Universal footer with dynamic links
│   ├── GlassyCard.js   # Glass morphism card component
│   ├── NetworkSelector.js # Network switching component
│   ├── PageLayout.js   # Main page layout wrapper
│   └── SideNavigation.js # Hierarchical navigation
├── contexts/           # React contexts
│   ├── NetworkContext.js # Network configuration management
│   └── RoleContext.js  # User role management
├── hooks/              # Custom React hooks
│   └── useNavigation.js # Navigation state management
├── pages/              # Page components
│   ├── dashboard.js    # Main dashboard
│   ├── models/         # Model-related pages
│   ├── marketplace/    # Marketplace pages
│   ├── vault/          # Personal asset management
│   └── explorers/      # Network explorers
└── styles/             # CSS modules and global styles
```

### Network Configuration
The WebGUI supports multiple network environments:

- **Mainnet**: Production KNIRV network
- **Testnet**: Testing environment with NRV tokens
- **Local**: Development environment
- **Private**: Private testing networks

### Backend Integration
- **Dynamic Backend Selection**: Switch between networks seamlessly
- **API Client**: Network-aware HTTP client with automatic endpoint switching
- **WebSocket Support**: Real-time data feeds
- **Health Monitoring**: Automatic network health checks

## 🛠️ Development

### Prerequisites
- Node.js 16+ 
- npm or yarn
- Access to KNIRV network endpoints

### Installation
```bash
cd KNIRVORACLE/services/webgui
npm install
```

### Development Server
```bash
npm run dev
```

### Build for Production
```bash
npm run build
```

### Testing
```bash
npm test
```

## 📱 Navigation Structure

### Main Sections
1. **Dashboard** - Network overview and quick actions
2. **Monitor** - Network monitoring and analytics
   - Network Monitor
   - Graph Explorer
   - Chain Explorer  
   - Oracle Explorer
   - Error Explorer
3. **Models** - AI model management
   - Codex Builder
   - Models DEX
   - KNIRVINFERENCE DAO
4. **Marketplace** - Trading platform
   - Skills
   - Capabilities
   - Properties
5. **Vault** - Personal asset management
   - My Models
   - My Skills
   - My Wallets
   - My Capabilities
   - My Properties
6. **Settings** - Configuration and preferences
7. **Network Admin** - Network administration (admin role)
8. **Auth Testing** - Authentication testing (dev mode)

## 🔧 Configuration

### Network Settings
Networks are configured in `src/contexts/NetworkContext.js`:

```javascript
const NETWORK_CONFIGS = {
  mainnet: {
    endpoints: {
      api: 'https://api.knirv.network',
      websocket: 'wss://ws.knirv.network',
      // ...
    }
  },
  // ...
}
```

### Environment Variables
Create a `.env.local` file:
```
REACT_APP_DEFAULT_NETWORK=testnet
REACT_APP_API_TIMEOUT=5000
REACT_APP_ENABLE_DEV_TOOLS=true
```

## 🎨 Styling

### Design System
- **Primary Color**: #007bff (KNIRV Blue)
- **Success Color**: #28a745 (Green)
- **Warning Color**: #ffc107 (Yellow)
- **Error Color**: #dc3545 (Red)
- **Glass Morphism**: Translucent backgrounds with blur effects
- **Responsive Breakpoints**: 768px, 480px

### CSS Modules
Each component uses CSS modules for scoped styling:
```javascript
import styles from './Component.module.css';
```

## 🔌 API Integration

### Network-Aware API Client
```javascript
import { useNetwork } from '../contexts/NetworkContext';

const { getApiClient } = useNetwork();
const api = getApiClient();

// Make requests
const data = await api.get('/models');
const result = await api.post('/models', modelData);
```

### WebSocket Connections
```javascript
const { currentNetwork } = useNetwork();
const ws = new WebSocket(currentNetwork.endpoints.websocket);
```

## 🧪 Testing

### Test Structure
```
src/
├── __tests__/          # Test files
│   ├── components/     # Component tests
│   ├── contexts/       # Context tests
│   ├── hooks/          # Hook tests
│   └── pages/          # Page tests
└── setupTests.js       # Test configuration
```

### Running Tests
```bash
# Run all tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm test -- --coverage
```

## 🚀 Deployment

### Build Process
1. Install dependencies: `npm install`
2. Build for production: `npm run build`
3. Deploy `build/` directory to web server
4. Configure reverse proxy for API endpoints

### Docker Deployment
```dockerfile
FROM node:16-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build
EXPOSE 3000
CMD ["npm", "start"]
```

## 🔒 Security

### Network Security
- HTTPS enforcement for production networks
- WebSocket secure connections (WSS)
- API key management
- CORS configuration

### Authentication
- Role-based access control
- JWT token management
- Session handling
- Guest mode support

## 📊 Monitoring

### Performance Metrics
- Network latency monitoring
- API response times
- WebSocket connection health
- Error rate tracking

### Health Checks
- Automatic network health monitoring
- Connection status indicators
- Fallback mechanisms
- Error recovery

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## 📄 License

This project is part of the KNIRV Network ecosystem. See the main repository for license information.

## 🆘 Support

For support and questions:
- Check the documentation
- Review existing issues
- Create a new issue with detailed information
- Contact the development team

---

**Note**: This WebGUI is designed to work with the broader KNIRV network ecosystem. Ensure you have access to the required network endpoints and services for full functionality.
