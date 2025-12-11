# KNIRV Developer Portal

The official developer portal for the KNIRV D-TEN (Decentralized Trusted Execution Network). This comprehensive web application provides developers with all the tools needed to build, test, and deploy autonomous agents on the KNIRV network.

## 🚀 Overview

The KNIRV Developer Portal is a static web application that serves as the primary interface for developers working with the KNIRV D-TEN network. It provides access to all nine layers of the KNIRV architecture and enables seamless development workflows.

### Key Features

- **🏠 Home Dashboard**: Quick overview and network statistics
- **📚 Core Concepts**: Understanding the D-TEN architecture
- **🎯 Getting Started**: Golden path tutorial for new developers
- **🤖 Agent Management**: Register and manage agents on KNIRV-CHAIN
- **🧠 Skill Registry**: Browse and publish skills from KNIRV-GRAPH
- **📜 UDC Management**: Issue User Delegation Certificates
- **🔄 Agent/Skill Exchange**: Trade ownership rights (no token trading)
- **🔍 ErrorNode Explorer**: Find upskilling opportunities
- **💻 API & SDK Reference**: Comprehensive documentation
- **💰 KNIRV Wallets**: NRN token and transaction management
- **🧪 TESNET & Sandbox**: Safe testing environments
- **👥 Community & Support**: Resources and community channels

## 🏗️ Architecture

The portal is built as a modern static web application using:

- **HTML5/CSS3/JavaScript**: Core web technologies
- **Tailwind CSS**: Utility-first CSS framework
- **Font Awesome**: Icon library
- **Responsive Design**: Mobile-first approach
- **Progressive Enhancement**: Works without JavaScript

### File Structure

```
developer-portal/
├── index.html              # Home Dashboard
├── core-concepts.html      # D-TEN Architecture Guide
├── getting-started.html    # Tutorial & Onboarding
├── agent-management.html   # Agent Registration & Management
├── skill-registry.html     # Skill Marketplace
├── udc-management.html     # User Delegation Certificates
├── agent-skill-exchange.html # Ownership Trading
├── error-node-explorer.html  # ErrorNode Discovery
├── api-sdk.html            # API Documentation
├── wallet-management.html  # NRN Token Management
├── tesnet-sandbox.html     # Testing Environments
├── community-support.html  # Community Resources
├── css/
│   └── portal.css          # Custom styles with KNIRV branding
└── js/
    ├── portal.js           # Core portal functionality
    └── udc-management.js   # UDC-specific features
```

## 🚀 Quick Start

### Local Development

1. **Serve the static files** (choose one method):

   ```bash
   # Using Python (recommended)
   npm run serve

   # Using Node.js http-server
   npm run serve-node

   # Using any static file server
   cd static && python3 -m http.server 8080
   ```

2. **Open your browser** and navigate to `http://localhost:8080`

### Testing

Run the integration tests to verify portal functionality:

```bash
# Run all portal tests
npm test

# Validate portal structure
npm run validate
```
## 🌐 Deployment

The portal is automatically deployed via Netlify when changes are pushed to the repository.

### Netlify Configuration

The portal is configured in the main `netlify.toml` file with:

- **Build Command**: No build needed (static files)
- **Publish Directory**: `developer-portal/`
- **Redirects**: Configured for `/portal/*`, `/developer/*`, `/dev-portal/*`

### Manual Deployment

To deploy to any static hosting service:

1. Copy the entire `static/` directory to your web server
2. Ensure proper MIME types are configured for `.html`, `.css`, `.js` files
3. Configure redirects if needed for SPA-like behavior

## 🔧 Development

### Adding New Pages

1. Create a new HTML file in the `static/` directory
2. Follow the existing template structure
3. Add navigation link to the sidebar in all pages
4. Update integration tests

### Customizing Styles

The portal uses a custom CSS framework built on Tailwind CSS with KNIRV branding:

- **Primary Color**: `#00c0fa` (KNIRV Blue)
- **Secondary Color**: `#2b56f5` (KNIRV Purple)
- **Glass Morphism**: Consistent across all components
- **Responsive Design**: Mobile-first approach

### JavaScript Functionality

- **Modular Architecture**: Each major feature has its own module
- **Event-Driven**: Uses modern event handling
- **Progressive Enhancement**: Core functionality works without JS
- **API Integration**: Mock APIs with realistic responses

## 🆕 Latest Updates

### Authentication System
- **🔐 Secure Login/Registration**: Modal-based authentication system
- **👤 User Profiles**: Role-based access control and permissions
- **🔒 Session Management**: Persistent login with localStorage
- **🎨 Integrated UI**: Seamless integration with existing portal design

### Dynamic Configuration Management
- **⚙️ YAML Configuration**: Centralized configuration in `../config/portal-links.yaml`
- **🔗 Dynamic Links**: All navigation and footer links sourced from configuration
- **🚩 Feature Flags**: Easy enable/disable of portal features
- **🔄 Live Updates**: Configuration changes apply without code modifications

### Universal Footer System
- **🌐 Consistent Branding**: Shared footer across all KNIRV Gateway pages
- **📱 Responsive Design**: Mobile-optimized footer layout
- **🔗 Configuration-Driven**: All links managed through central configuration
- **🎨 KNIRV Styling**: Matches the overall portal design system

### Enhanced Navigation
- **🖼️ iFrame Integration**: Seamless access to Graphchain Explorer, Documentation, and KNIRV-NEXUS
- **🏠 Main Site Links**: Easy navigation back to KNIRV.com
- **💳 Payment Integration**: Direct links to KNIRV payment gateway
- **📚 Support Integration**: Direct access to support desk and documentation

### TESNET & Sandbox Improvements
- **🏗️ CDE Pitch Integration**: Information about Collaborative Development Environment
- **🔗 KNIRV-NEXUS Integration**: Direct access to DVE (Decentralized Validation Environment)
- **📖 Enhanced Documentation**: Detailed information about secure execution environments
- **🎯 Improved UX**: Better user guidance for sandbox creation

### Files Added/Modified
- `js/auth.js` - Complete authentication system
- `js/config-loader.js` - Configuration management system
- `../js/universal-footer.js` - Shared footer component
- `../config/portal-links.yaml` - Master configuration file
- `../config/portal-config.json` - Browser-compatible configuration
- `AUTHENTICATION_GUIDE.md` - Complete authentication documentation
- `../CONFIGURATION_MANAGEMENT.md` - Configuration system guide

## 🧪 Testing

### Integration Tests

Located in `../integration-tests/`:

- **Portal Integration**: Tests navigation, links, and basic functionality
- **Validation**: Checks file structure and required elements
- **Performance**: Basic performance and accessibility checks

### Manual Testing Checklist

- [ ] All navigation links work correctly
- [ ] Forms submit and show appropriate feedback
- [ ] Responsive design works on mobile/tablet/desktop
- [ ] All interactive elements function properly
- [ ] Error states are handled gracefully

## 🔗 Integration

### Main Website Integration

The portal is integrated with the main KNIRV website:

- **Navigation Links**: Added to main site header and banner
- **Consistent Branding**: Matches main site design language
- **Seamless Transitions**: Smooth navigation between site and portal

### API Integration

The portal includes mock API integrations for:

- **KNIRV-CHAIN**: Agent registration and credential management
- **KNIRV-GRAPH**: Skill publishing and ErrorNode exploration
- **KNIRV-NEXUS**: Secure execution environment
- **KNIRV-WALLET**: Token management and transactions

## 📚 Documentation

- **Core Concepts**: Built-in guide to D-TEN architecture
- **API Reference**: Interactive API explorer
- **Getting Started**: Step-by-step tutorial
- **Community Resources**: Links to external documentation

## 🤝 Contributing

1. Follow the existing code structure and naming conventions
2. Test all changes thoroughly
3. Update documentation as needed
4. Ensure responsive design compatibility

## 📄 License

This project is part of the KNIRV Network ecosystem. See the main repository for license information.

## 🆘 Support

- **Community Discord**: Real-time support and discussions
- **GitHub Issues**: Bug reports and feature requests
- **Documentation**: Comprehensive guides and API references
- **Support Portal**: Direct access to the development team

---

**Built with ❤️ for the KNIRV D-TEN Network**