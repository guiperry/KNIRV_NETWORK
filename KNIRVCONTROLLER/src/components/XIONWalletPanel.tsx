/**
 * XION Wallet Panel Component
 * Provides UI for XION Meta Accounts connection and USDC-to-NRN conversions
 * Integrates with KNIRVORACLE Payment Gateway
 */

import React, { useState, useEffect, useCallback } from 'react';
import useXIONWallet from '../hooks/useXIONWallet';
import { ConversionRequest, ConversionResult } from '../services/AbstraxionWalletService';

interface XIONWalletPanelProps {
  className?: string;
}

export const XIONWalletPanel: React.FC<XIONWalletPanelProps> = ({ className = '' }) => {
  const {
    account,
    isConnected,
    isConnecting,
    error,
    paymentGatewayConfig,
    conversionRates,
    connectWallet,
    disconnectWallet,
    convertUSDCToNRN,
    checkConversionStatus,
    getPaymentHistory,
    refreshBalance,
    refreshRates,
    connectMetaAccount,
  } = useXIONWallet();

  // Local state
  const [conversionAmount, setConversionAmount] = useState('');
  const [conversionHistory, setConversionHistory] = useState<ConversionResult[]>([]);
  const [activeConversions, setActiveConversions] = useState<ConversionResult[]>([]);
  const [selectedAuthMethod, setSelectedAuthMethod] = useState<'email' | 'social' | 'wallet' | 'passkey'>('email');
  const [metaAccountIdentifier, setMetaAccountIdentifier] = useState('');

  // Load payment history
  const loadPaymentHistory = useCallback(async () => {
    try {
      const history = await getPaymentHistory();
      setConversionHistory(history);
    } catch (error) {
      console.error('Failed to load payment history:', error);
    }
  }, [getPaymentHistory]);

  // Load payment history when connected
  useEffect(() => {
    if (isConnected) {
      loadPaymentHistory();
    }
  }, [isConnected, loadPaymentHistory]);

  // Handle wallet connection
  const handleConnect = async () => {
    try {
      await connectWallet(selectedAuthMethod);
    } catch (error) {
      console.error('Connection failed:', error);
    }
  };

  // Handle USDC to NRN conversion
  const handleConversion = async () => {
    if (!conversionAmount || parseFloat(conversionAmount) <= 0) {
      alert('Please enter a valid amount');
      return;
    }

    try {
      const request: ConversionRequest = {
        usdcAmount: (parseFloat(conversionAmount) * 1000000).toString(), // Convert to micro USDC
        gasless: true,
        memo: 'USDC to NRN conversion via KNIRVCONTROLLER'
      };

      const result = await convertUSDCToNRN(request);
      
      // Add to active conversions for monitoring
      setActiveConversions(prev => [...prev, result]);
      
      // Clear input
      setConversionAmount('');
      
      // Monitor the conversion
      monitorConversion(result.transactionId);
      
      alert(`Conversion initiated! Transaction ID: ${result.transactionId}`);
    } catch (error) {
      console.error('Conversion failed:', error);
      alert(`Conversion failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  // Monitor conversion status
  const monitorConversion = async (transactionId: string) => {
    const checkStatus = async () => {
      try {
        const status = await checkConversionStatus(transactionId);
        
        if (status) {
          setActiveConversions(prev => 
            prev.map(conv => 
              conv.transactionId === transactionId ? status : conv
            )
          );

          if (status.status === 'confirmed') {
            // Move to history and refresh balance
            await loadPaymentHistory();
            await refreshBalance();
            
            // Remove from active conversions
            setActiveConversions(prev => 
              prev.filter(conv => conv.transactionId !== transactionId)
            );
          } else if (status.status === 'pending') {
            // Check again in 5 seconds
            setTimeout(checkStatus, 5000);
          }
        }
      } catch (error) {
        console.error('Failed to check conversion status:', error);
      }
    };

    // Start monitoring
    setTimeout(checkStatus, 2000);
  };

  // Handle meta account connection
  const handleMetaAccountConnect = async () => {
    if (!metaAccountIdentifier) {
      alert('Please enter an identifier');
      return;
    }

    try {
      const result = await connectMetaAccount(selectedAuthMethod, metaAccountIdentifier);
      alert(`Meta account connected: ${result.account || 'Unknown'}`);
      setMetaAccountIdentifier('');
    } catch (error) {
      console.error('Meta account connection failed:', error);
      alert(`Connection failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  // Calculate NRN amount from USDC
  const calculateNRNAmount = () => {
    if (!conversionAmount || !conversionRates) return '0';
    const usdcAmount = parseFloat(conversionAmount);
    const rate = parseFloat(
      (conversionRates as { usdc_to_nrn?: string })?.usdc_to_nrn || '0'
    );
    return (usdcAmount * rate).toFixed(2);
  };

  return (
    <div className={`xion-wallet-panel ${className}`}>
      <div className="panel-header">
        <h2>XION Wallet & Payment Gateway</h2>
        <p>Seamless USDC to NRN conversions with Meta Accounts</p>
      </div>

      {error && (
        <div className="error-message">
          <strong>Error:</strong> {error}
        </div>
      )}

      {!isConnected ? (
        <div className="connection-section">
          <h3>Connect XION Wallet</h3>
          
          <div className="auth-method-selector">
            <label>Authentication Method:</label>
            <select 
              value={selectedAuthMethod} 
              onChange={(e) => setSelectedAuthMethod(e.target.value as 'email' | 'social' | 'wallet' | 'passkey')}
            >
              <option value="email">Email</option>
              <option value="social">Social</option>
              <option value="wallet">Wallet</option>
              <option value="passkey">Passkey</option>
            </select>
          </div>

          <button 
            onClick={handleConnect} 
            disabled={isConnecting}
            className="connect-button"
          >
            {isConnecting ? 'Connecting...' : `Connect with ${selectedAuthMethod}`}
          </button>

          <div className="meta-account-section">
            <h4>Or Connect Meta Account</h4>
            <input
              type="text"
              placeholder={`Enter ${selectedAuthMethod} identifier`}
              value={metaAccountIdentifier}
              onChange={(e) => setMetaAccountIdentifier(e.target.value)}
            />
            <button onClick={handleMetaAccountConnect}>
              Connect Meta Account
            </button>
          </div>
        </div>
      ) : (
        <div className="wallet-connected">
          <div className="account-info">
            <h3>Connected Account</h3>
            <p><strong>Address:</strong> {account?.address}</p>
            <p><strong>Type:</strong> {account?.metaAccountType}</p>
            <p><strong>Gasless:</strong> {account?.gasless ? 'Enabled' : 'Disabled'}</p>
            
            <div className="balances">
              <div className="balance-item">
                <span>USDC:</span>
                <span>{account ? (parseFloat(account.usdcBalance) / 1000000).toFixed(2) : '0'}</span>
              </div>
              <div className="balance-item">
                <span>NRN:</span>
                <span>{account ? (parseFloat(account.nrnBalance) / 1e18).toFixed(2) : '0'}</span>
              </div>
            </div>

            <button onClick={refreshBalance} className="refresh-button">
              Refresh Balance
            </button>
            <button onClick={disconnectWallet} className="disconnect-button">
              Disconnect
            </button>
          </div>

          <div className="conversion-section">
            <h3>Convert USDC to NRN</h3>
            
            {(conversionRates && typeof conversionRates === 'object' && conversionRates !== null && (
              <div className="conversion-rate">
                <p>Rate: 1 USDC = {(conversionRates as { usdc_to_nrn?: string })?.usdc_to_nrn || '0'} NRN</p>
                <button
                  onClick={refreshRates}
                  className="refresh-rates-btn"
                  title="Refresh exchange rates"
                >
                  🔄
                </button>
              </div>
            )) as React.ReactNode}

            <div className="conversion-input">
              <input
                type="number"
                placeholder="Enter USDC amount"
                value={conversionAmount}
                onChange={(e) => setConversionAmount(e.target.value)}
                min="0"
                step="0.01"
              />
              <span className="conversion-preview">
                ≈ {calculateNRNAmount()} NRN
              </span>
            </div>

            <button 
              onClick={handleConversion}
              disabled={!conversionAmount || parseFloat(conversionAmount) <= 0}
              className="convert-button"
            >
              Convert to NRN (Gasless)
            </button>
          </div>

          {activeConversions.length > 0 && (
            <div className="active-conversions">
              <h3>Active Conversions</h3>
              {activeConversions.map((conversion) => (
                <div key={conversion.transactionId} className="conversion-item">
                  <p><strong>ID:</strong> {conversion.transactionId}</p>
                  <p><strong>Amount:</strong> {(parseFloat(conversion.usdcAmount) / 1000000).toFixed(2)} USDC → {(parseFloat(conversion.nrnAmount) / 1e18).toFixed(2)} NRN</p>
                  <p><strong>Status:</strong> {conversion.status}</p>
                </div>
              ))}
            </div>
          )}

          {conversionHistory.length > 0 && (
            <div className="conversion-history">
              <h3>Conversion History</h3>
              {conversionHistory.slice(0, 5).map((conversion) => (
                <div key={conversion.transactionId} className="history-item">
                  <p><strong>Date:</strong> {conversion.timestamp.toLocaleDateString()}</p>
                  <p><strong>Amount:</strong> {(parseFloat(conversion.usdcAmount) / 1000000).toFixed(2)} USDC → {(parseFloat(conversion.nrnAmount) / 1e18).toFixed(2)} NRN</p>
                  <p><strong>Status:</strong> {conversion.status}</p>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {(paymentGatewayConfig && typeof paymentGatewayConfig === 'object' && paymentGatewayConfig !== null && (
        <div className="gateway-info">
          <h4>Payment Gateway Info</h4>
          <p><strong>Chain:</strong> {(paymentGatewayConfig as { chain_id?: string })?.chain_id || 'Unknown'}</p>
          <p><strong>Gasless:</strong> {(paymentGatewayConfig as { gasless_enabled?: boolean })?.gasless_enabled ? 'Enabled' : 'Disabled'}</p>
          <p><strong>Min Amount:</strong> {(parseFloat((paymentGatewayConfig as { min_transaction_amount?: string })?.min_transaction_amount || '0') / 1000000).toFixed(2)} USDC</p>
          <p><strong>Max Amount:</strong> {(parseFloat((paymentGatewayConfig as { max_transaction_amount?: string })?.max_transaction_amount || '0') / 1000000).toFixed(0)} USDC</p>
        </div>
      )) as React.ReactNode}
    </div>
  );
};

export default XIONWalletPanel;
