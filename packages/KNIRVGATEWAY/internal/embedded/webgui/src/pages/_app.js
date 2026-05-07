import React from 'react';
import '../styles/globals.css';
import RoleProtectedRoute from '../components/RoleProtectedRoute';
import { BackendProvider } from '../contexts/BackendContext';
import { BlockchainProvider } from '../contexts/BlockchainContext';
import { RoleProvider } from '../contexts/RoleContext';

// Global styles are now in globals.css

function MyApp({ Component, pageProps }) {
  return (
    <BackendProvider>
      <BlockchainProvider>
        <RoleProvider>
          <RoleProtectedRoute>
            <Component {...pageProps} />
          </RoleProtectedRoute>
        </RoleProvider>
      </BlockchainProvider>
    </BackendProvider>
  );
}

export default MyApp;