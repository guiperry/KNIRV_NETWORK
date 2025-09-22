# KNIRVGATEWAY WebGUI Authentication Testing Guide

## Overview

The WebGUI now includes comprehensive authentication testing tools that allow you to test all user roles, authentication scenarios, and access controls without needing to set up complex authentication flows.

## New Features Added

### 1. Role Switcher (Top-Right Corner)
- **Location**: Fixed position in top-right corner of all pages
- **Features**: 
  - Switch between Root, Bootnode, Dev, and General roles instantly
  - Change network contexts (Mainnet, Public Testnet, Private Testnet, Demo)
  - View current authentication state
  - Clear authentication to test login screen

### 2. Authentication Testing Page
- **Access**: Navigate to "Auth Testing" in the sidebar (🔐 icon)
- **Features**:
  - Real-time authentication status display
  - Page access testing for current role
  - Role permission comparison matrix
  - Authentication history tracking
  - Comprehensive testing instructions

## How to Test Different Roles

### Testing Root Role
1. Open WebGUI at http://localhost:3007
2. Click the role switcher in the top-right corner
3. Click "Root" button
4. Page will reload with Root permissions
5. **Expected Access**: All pages including Network Admin

### Testing Bootnode Role
1. Use role switcher to select "Bootnode"
2. **Expected Access**: Most pages except Network Admin

### Testing Developer Role
1. Use role switcher to select "Developer"
2. **Expected Access**: Development-focused pages, no admin features

### Testing General Role
1. Use role switcher to select "General"
2. **Expected Access**: Basic user pages only

## How to Test Login Screen

### Method 1: Clear Authentication
1. Click role switcher in top-right
2. Click "Clear Auth & Show Login" button
3. Page reloads and shows authentication screen
4. You'll see "Authentication Required" with options:
   - "Go to Main Website" - Redirects to main site
   - "Demo Mode" - Enables demo access (development only)

### Method 2: Manual Clear
1. Open browser developer tools (F12)
2. Go to Application/Storage tab
3. Clear localStorage items:
   - `knirv_user_role`
   - `knirv_network`
   - `knirv_auth_token`
   - `knirv_demo_mode`
4. Refresh page

## Testing Page Access Controls

### Using the Auth Testing Page
1. Navigate to "Auth Testing" in sidebar
2. View "Page Access Testing" section
3. **Green buttons**: Pages your current role can access
4. **Red buttons**: Pages denied to your current role
5. Click buttons to test navigation (denied pages should be blocked)

### Manual Testing
1. Try navigating to restricted pages directly:
   - Root only: `/network-admin`
   - Bootnode+: `/peers`, `/settlement`
   - Dev+: `/blockchain`, `/vault`
   - All roles: `/dashboard`, `/inventory`

## Role Permission Matrix

| Page | Root | Bootnode | Dev | General |
|------|------|----------|-----|---------|
| Dashboard | ✅ | ✅ | ✅ | ✅ |
| Inventory | ✅ | ✅ | ✅ | ✅ |
| DEX | ✅ | ✅ | ✅ | ✅ |
| NFT Capability Manager | ✅ | ✅ | ✅ | ✅ |
| Capabilities | ✅ | ✅ | ✅ | ✅ |
| Auth Testing | ✅ | ✅ | ✅ | ✅ |
| Vault | ✅ | ✅ | ✅ | ❌ |
| Blockchain | ✅ | ✅ | ✅ | ❌ |
| NFT Vault | ✅ | ✅ | ✅ | ❌ |
| Add Capability | ✅ | ✅ | ✅ | ❌ |
| Explorer | ✅ | ✅ | ✅ | ❌ |
| DAOs | ✅ | ✅ | ❌ | ❌ |
| Settlement | ✅ | ✅ | ❌ | ❌ |
| Peers | ✅ | ✅ | ❌ | ❌ |
| Network Admin | ✅ | ❌ | ❌ | ❌ |

## Testing Network Contexts

### Available Networks
1. **Demo Mode**: Local testing environment
2. **Private Testnet**: Internal testing network
3. **Public Testnet**: External testing network
4. **Mainnet**: Production network

### How to Test
1. Use role switcher to change networks
2. Observe network display in status areas
3. Check if different networks affect available features

## Troubleshooting

### Role Switcher Not Visible
- Ensure you're on a page that includes the PageLayout component
- Check browser console for JavaScript errors
- Try refreshing the page

### Authentication Not Working
1. Clear browser cache and localStorage
2. Restart the KNIRVGATEWAY services: `npm run services:start`
3. Check that backend is running on port 8080
4. Verify webgui service is running on port 3007

### Page Access Issues
1. Check current role in role switcher
2. Verify page is in the allowed list for your role
3. Check browser console for navigation errors
4. Try switching to a higher privilege role

## Development Notes

### For Developers
- Role switcher is available in development mode only
- Demo mode bypasses normal authentication requirements
- Authentication state is stored in localStorage
- Page access is controlled by the RoleContext

### For Production
- Role switcher should be hidden in production builds
- Authentication must come from main KNIRV website
- Demo mode should be disabled
- All authentication tokens should be properly validated

## Quick Test Scenarios

### Scenario 1: Full Role Testing
1. Start as General → Test basic access
2. Switch to Dev → Test development features
3. Switch to Bootnode → Test network features
4. Switch to Root → Test admin features

### Scenario 2: Authentication Flow Testing
1. Clear auth → See login screen
2. Enable demo mode → Get General access
3. Switch roles → Test different permissions
4. Logout → Return to login screen

### Scenario 3: Page Access Testing
1. Go to Auth Testing page
2. Test all green buttons (should work)
3. Test all red buttons (should be blocked)
4. Switch roles and repeat

This testing system provides comprehensive coverage of all authentication scenarios without requiring complex setup or external authentication services.
