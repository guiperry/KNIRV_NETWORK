# KNIRV Wallet Torus Sign-In Module

A specialized authentication module that integrates Web3Auth (formerly Torus) social login capabilities into the KNIRV Wallet ecosystem, enabling users to create and access wallets using their existing social media accounts.

## Overview

The KNIRV Wallet Torus Sign-In Module (`knirvwallet-torus-signin`) provides:
- **Social Login Integration**: Login with Google, Facebook, Twitter, Discord, and more
- **Non-Custodial Security**: Users maintain control of their private keys
- **Seamless Onboarding**: Simplified wallet creation for mainstream users
- **Multi-Factor Authentication**: Enhanced security through social providers
- **Cross-Platform Compatibility**: Works across web, mobile, and extension environments

## Features

### 🔐 Social Authentication Providers
- **Google**: Gmail and Google account integration
- **Facebook**: Facebook account authentication
- **Twitter**: Twitter/X account login
- **Discord**: Discord community integration
- **GitHub**: Developer-friendly authentication
- **LinkedIn**: Professional network integration
- **Apple**: iOS and macOS native integration
- **Custom OAuth**: Support for custom OAuth providers

### 🛡️ Security Features
- **Non-Custodial**: Private keys are generated and stored locally
- **Threshold Cryptography**: Distributed key management
- **Multi-Party Computation**: Secure key operations
- **Biometric Authentication**: Device-level security integration
- **Session Management**: Secure session handling and refresh

### 🌐 Cross-Platform Support
- **Web Browsers**: Chrome, Firefox, Safari, Edge
- **Mobile Apps**: React Native integration
- **Browser Extensions**: Seamless extension integration
- **Desktop Apps**: Electron and native app support

## Installation

```bash
# Using npm
npm install knirvwallet-torus-signin

# Using yarn
yarn add knirvwallet-torus-signin

# Using pnpm
pnpm add knirvwallet-torus-signin
```

## Quick Start

### Basic Setup

```typescript
import { TorusSignIn } from 'knirvwallet-torus-signin';

// Initialize Torus Sign-In
const torusSignIn = new TorusSignIn({
  clientId: 'your-web3auth-client-id',
  network: 'mainnet', // or 'testnet'
  chainConfig: {
    chainNamespace: 'other',
    chainId: 'knirv-1',
    rpcTarget: 'https://rpc.knirv.network',
    displayName: 'KNIRV Network',
    blockExplorer: 'https://explorer.knirv.network',
    ticker: 'NRN',
    tickerName: 'KNIRV Network Token'
  }
});

// Initialize the service
await torusSignIn.init();
```

### Social Login

```typescript
// Login with Google
const googleUser = await torusSignIn.loginWithGoogle();
console.log('User info:', googleUser);
console.log('Wallet address:', googleUser.walletAddress);
console.log('Private key:', googleUser.privateKey);

// Login with Facebook
const facebookUser = await torusSignIn.loginWithFacebook();

// Login with Twitter
const twitterUser = await torusSignIn.loginWithTwitter();

// Login with Discord
const discordUser = await torusSignIn.loginWithDiscord();
```

### Custom Provider Login

```typescript
// Login with custom OAuth provider
const customUser = await torusSignIn.loginWithCustomProvider({
  provider: 'custom-oauth',
  clientId: 'your-custom-client-id',
  redirectUri: 'https://your-app.com/callback'
});
```

## API Reference

### `TorusSignIn` Class

#### Constructor Options

```typescript
interface TorusSignInOptions {
  clientId: string;
  network: 'mainnet' | 'testnet' | 'development';
  chainConfig: ChainConfig;
  uiConfig?: UIConfig;
  storageKey?: string;
  enableLogging?: boolean;
}

interface ChainConfig {
  chainNamespace: string;
  chainId: string;
  rpcTarget: string;
  displayName: string;
  blockExplorer: string;
  ticker: string;
  tickerName: string;
}

interface UIConfig {
  theme: 'light' | 'dark' | 'auto';
  appLogo?: string;
  appName?: string;
  defaultLanguage?: string;
}
```

#### Methods

```typescript
class TorusSignIn {
  // Initialize the service
  async init(): Promise<void>;
  
  // Check if user is logged in
  isLoggedIn(): boolean;
  
  // Get current user info
  getUserInfo(): Promise<UserInfo | null>;
  
  // Social login methods
  async loginWithGoogle(): Promise<UserInfo>;
  async loginWithFacebook(): Promise<UserInfo>;
  async loginWithTwitter(): Promise<UserInfo>;
  async loginWithDiscord(): Promise<UserInfo>;
  async loginWithGitHub(): Promise<UserInfo>;
  async loginWithLinkedIn(): Promise<UserInfo>;
  async loginWithApple(): Promise<UserInfo>;
  
  // Custom provider login
  async loginWithCustomProvider(config: CustomProviderConfig): Promise<UserInfo>;
  
  // Logout
  async logout(): Promise<void>;
  
  // Get wallet instance
  getWallet(): Promise<TorusWallet>;
  
  // Sign transaction
  async signTransaction(transaction: Transaction): Promise<string>;
  
  // Sign message
  async signMessage(message: string): Promise<string>;
}
```

#### User Info Interface

```typescript
interface UserInfo {
  email: string;
  name: string;
  profileImage: string;
  aggregateVerifier: string;
  verifier: string;
  verifierId: string;
  typeOfLogin: string;
  walletAddress: string;
  privateKey: string;
  publicKey: string;
  dappShare: string;
  idToken: string;
  oAuthIdToken: string;
  oAuthAccessToken: string;
}
```

## Integration Examples

### React Web Application

```typescript
import React, { useState, useEffect } from 'react';
import { TorusSignIn } from 'knirvwallet-torus-signin';

function WalletLogin() {
  const [torusSignIn, setTorusSignIn] = useState<TorusSignIn | null>(null);
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const initTorus = async () => {
      const torus = new TorusSignIn({
        clientId: process.env.REACT_APP_WEB3AUTH_CLIENT_ID!,
        network: 'mainnet',
        chainConfig: {
          chainNamespace: 'other',
          chainId: 'knirv-1',
          rpcTarget: 'https://rpc.knirv.network',
          displayName: 'KNIRV Network',
          blockExplorer: 'https://explorer.knirv.network',
          ticker: 'NRN',
          tickerName: 'KNIRV Network Token'
        }
      });
      
      await torus.init();
      setTorusSignIn(torus);
      
      // Check if already logged in
      if (torus.isLoggedIn()) {
        const userInfo = await torus.getUserInfo();
        setUser(userInfo);
      }
    };

    initTorus();
  }, []);

  const handleGoogleLogin = async () => {
    if (!torusSignIn) return;
    
    setLoading(true);
    try {
      const userInfo = await torusSignIn.loginWithGoogle();
      setUser(userInfo);
    } catch (error) {
      console.error('Login failed:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    if (!torusSignIn) return;
    
    await torusSignIn.logout();
    setUser(null);
  };

  if (!torusSignIn) {
    return <div>Initializing...</div>;
  }

  if (user) {
    return (
      <div>
        <h2>Welcome, {user.name}!</h2>
        <p>Wallet Address: {user.walletAddress}</p>
        <button onClick={handleLogout}>Logout</button>
      </div>
    );
  }

  return (
    <div>
      <h2>Login to KNIRV Wallet</h2>
      <button onClick={handleGoogleLogin} disabled={loading}>
        {loading ? 'Logging in...' : 'Login with Google'}
      </button>
    </div>
  );
}

export default WalletLogin;
```

### React Native Integration

```typescript
import React, { useState, useEffect } from 'react';
import { View, Text, TouchableOpacity, Alert } from 'react-native';
import { TorusSignIn } from 'knirvwallet-torus-signin';

const WalletScreen = () => {
  const [torusSignIn, setTorusSignIn] = useState<TorusSignIn | null>(null);
  const [user, setUser] = useState<UserInfo | null>(null);

  useEffect(() => {
    const initTorus = async () => {
      try {
        const torus = new TorusSignIn({
          clientId: 'your-client-id',
          network: 'mainnet',
          chainConfig: {
            chainNamespace: 'other',
            chainId: 'knirv-1',
            rpcTarget: 'https://rpc.knirv.network',
            displayName: 'KNIRV Network',
            blockExplorer: 'https://explorer.knirv.network',
            ticker: 'NRN',
            tickerName: 'KNIRV Network Token'
          }
        });
        
        await torus.init();
        setTorusSignIn(torus);
      } catch (error) {
        Alert.alert('Error', 'Failed to initialize wallet');
      }
    };

    initTorus();
  }, []);

  const handleSocialLogin = async (provider: string) => {
    if (!torusSignIn) return;

    try {
      let userInfo;
      switch (provider) {
        case 'google':
          userInfo = await torusSignIn.loginWithGoogle();
          break;
        case 'facebook':
          userInfo = await torusSignIn.loginWithFacebook();
          break;
        case 'twitter':
          userInfo = await torusSignIn.loginWithTwitter();
          break;
        default:
          throw new Error('Unsupported provider');
      }
      
      setUser(userInfo);
      Alert.alert('Success', `Welcome ${userInfo.name}!`);
    } catch (error) {
      Alert.alert('Error', 'Login failed');
    }
  };

  return (
    <View style={{ flex: 1, padding: 20 }}>
      {user ? (
        <View>
          <Text>Welcome, {user.name}!</Text>
          <Text>Address: {user.walletAddress}</Text>
          <TouchableOpacity onPress={() => torusSignIn?.logout()}>
            <Text>Logout</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <View>
          <TouchableOpacity onPress={() => handleSocialLogin('google')}>
            <Text>Login with Google</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => handleSocialLogin('facebook')}>
            <Text>Login with Facebook</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => handleSocialLogin('twitter')}>
            <Text>Login with Twitter</Text>
          </TouchableOpacity>
        </View>
      )}
    </View>
  );
};

export default WalletScreen;
```

### Browser Extension Integration

```typescript
// In background script
import { TorusSignIn } from 'knirvwallet-torus-signin';

class ExtensionWalletManager {
  private torusSignIn: TorusSignIn;

  constructor() {
    this.torusSignIn = new TorusSignIn({
      clientId: 'extension-client-id',
      network: 'mainnet',
      chainConfig: {
        chainNamespace: 'other',
        chainId: 'knirv-1',
        rpcTarget: 'https://rpc.knirv.network',
        displayName: 'KNIRV Network',
        blockExplorer: 'https://explorer.knirv.network',
        ticker: 'NRN',
        tickerName: 'KNIRV Network Token'
      }
    });
  }

  async init() {
    await this.torusSignIn.init();
  }

  async handleSocialLogin(provider: string) {
    switch (provider) {
      case 'google':
        return await this.torusSignIn.loginWithGoogle();
      case 'facebook':
        return await this.torusSignIn.loginWithFacebook();
      default:
        throw new Error('Unsupported provider');
    }
  }

  async signTransaction(transaction: any) {
    return await this.torusSignIn.signTransaction(transaction);
  }
}

// Initialize in background
const walletManager = new ExtensionWalletManager();
walletManager.init();
```

## Configuration

### Environment Variables

```bash
# Web3Auth Configuration
WEB3AUTH_CLIENT_ID=your_web3auth_client_id
WEB3AUTH_NETWORK=mainnet

# KNIRV Network Configuration
KNIRV_RPC_URL=https://rpc.knirv.network
KNIRV_CHAIN_ID=knirv-1
KNIRV_EXPLORER_URL=https://explorer.knirv.network
```

### Advanced Configuration

```typescript
const torusSignIn = new TorusSignIn({
  clientId: 'your-client-id',
  network: 'mainnet',
  chainConfig: {
    chainNamespace: 'other',
    chainId: 'knirv-1',
    rpcTarget: 'https://rpc.knirv.network',
    displayName: 'KNIRV Network',
    blockExplorer: 'https://explorer.knirv.network',
    ticker: 'NRN',
    tickerName: 'KNIRV Network Token'
  },
  uiConfig: {
    theme: 'dark',
    appLogo: 'https://your-app.com/logo.png',
    appName: 'Your App Name',
    defaultLanguage: 'en'
  },
  storageKey: 'knirv_wallet_torus',
  enableLogging: true
});
```

## Security Considerations

### Best Practices
- **Client ID Security**: Keep your Web3Auth client ID secure
- **Network Validation**: Always validate network configurations
- **Session Management**: Implement proper session timeout handling
- **Error Handling**: Handle authentication errors gracefully

### Security Features
- **Non-Custodial**: Users control their private keys
- **Threshold Signatures**: Distributed key management
- **Secure Storage**: Encrypted local storage
- **Session Security**: Secure session management

## Troubleshooting

### Common Issues

#### Authentication Failures
- Verify Web3Auth client ID is correct
- Check network configuration
- Ensure popup blockers are disabled

#### Network Issues
- Verify RPC endpoint is accessible
- Check chain ID configuration
- Ensure proper CORS settings

#### Mobile Integration Issues
- Configure proper URL schemes
- Handle deep linking correctly
- Test on different devices

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Note**: This module provides social authentication for the KNIRV Wallet ecosystem. For complete wallet functionality, integrate with the [core module](../knirvwallet-module/README.md) and [browser extension](../knirvwallet-extension/README.md).
