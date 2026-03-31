/**
 * USDC to NRN Purchase Component
 * Implements seamless USDC to NRN token purchase flow using XION Meta Accounts
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useAbstraxionWallet, ConversionRequest, ConversionResult } from '../services/AbstraxionWalletService';

interface USDCToNRNPurchaseProps {
  onPurchaseComplete?: (result: ConversionResult) => void;
  onError?: (error: Error) => void;
}

export const USDCToNRNPurchase: React.FC<USDCToNRNPurchaseProps> = ({
  onPurchaseComplete,
  onError
}) => {
  const {
    account,
    isConnected,
    isLoading,
    connect,
    disconnect,
    convertUSDCToNRN,
    getUSDCBalance
  } = useAbstraxionWallet();

  const [usdcAmount, setUsdcAmount] = useState('');
  const [nrnTargetAddress, setNrnTargetAddress] = useState('');
  const [memo, setMemo] = useState('');
  const [authMethod, setAuthMethod] = useState<'email' | 'social' | 'wallet' | 'passkey'>('email');
  const [conversionHistory, setConversionHistory] = useState<ConversionResult[]>([]);
  const [isConverting, setIsConverting] = useState(false);

  // Conversion rate (in real implementation, this would come from an oracle)
  const conversionRate = 10; // 1 USDC = 10 NRN
  const estimatedNRN = usdcAmount ? (parseFloat(usdcAmount) * conversionRate).toString() : '0';

  const loadConversionHistory = useCallback(async () => {
    try {
      // TODO: Implement getConversionHistory in useXIONWallet
      // const history = await getConversionHistory();
      // setConversionHistory(history);
      setConversionHistory([]);
    } catch (error) {
      console.error('Failed to load conversion history:', error);
    }
  }, []);

  useEffect(() => {
    if (isConnected) {
      loadConversionHistory();
    }
  }, [isConnected, loadConversionHistory]);

  const handleConnect = async () => {
    try {
      await connect(authMethod);
    } catch (error) {
      console.error('Connection failed:', error);
      onError?.(error as Error);
    }
  };

  const handleDisconnect = async () => {
    try {
      await disconnect();
      setConversionHistory([]);
    } catch (error) {
      console.error('Disconnection failed:', error);
    }
  };

  const handlePurchase = async () => {
    if (!account || !usdcAmount) return;

    setIsConverting(true);
    try {
      const request: ConversionRequest = {
        usdcAmount,
        nrnTargetAddress: nrnTargetAddress || account.address,
        memo: memo || 'USDC to NRN conversion via KNIRV Controller',
        gasless: true // Enable gasless transactions by default
      };

      const result = await convertUSDCToNRN(request);
      
      // Reset form
      setUsdcAmount('');
      setNrnTargetAddress('');
      setMemo('');
      
      // Reload history
      await loadConversionHistory();
      
      onPurchaseComplete?.(result);
    } catch (error) {
      console.error('Purchase failed:', error);
      onError?.(error as Error);
    } finally {
      setIsConverting(false);
    }
  };

  const isValidAmount = usdcAmount && parseFloat(usdcAmount) > 0 && 
    account && parseFloat(usdcAmount) <= parseFloat(account.usdcBalance);

  if (!isConnected) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6 max-w-md mx-auto">
        <h2 className="text-2xl font-bold text-gray-800 mb-6">Connect XION Wallet</h2>
        
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Authentication Method
          </label>
          <select
            value={authMethod}
            onChange={(e) => setAuthMethod(e.target.value as 'email' | 'social' | 'wallet' | 'passkey')}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="email">Email</option>
            <option value="social">Social Login</option>
            <option value="wallet">Wallet Connect</option>
            <option value="passkey">Passkey</option>
          </select>
        </div>

        <button
          onClick={handleConnect}
          disabled={isLoading}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isLoading ? 'Connecting...' : `Connect with ${authMethod}`}
        </button>

        <div className="mt-4 text-sm text-gray-600">
          <p>Connect your XION Meta Account to purchase NRN tokens with USDC.</p>
          <p className="mt-2">✨ Gasless transactions enabled via Treasury Contract</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6 max-w-md mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-800">Purchase NRN Tokens</h2>
        <button
          onClick={handleDisconnect}
          className="text-sm text-gray-500 hover:text-gray-700"
        >
          Disconnect
        </button>
      </div>

      {/* Account Info */}
      {account && (
        <div className="bg-gray-50 rounded-lg p-4 mb-6">
          <div className="text-sm text-gray-600 mb-2">
            <span className="font-medium">Account:</span> {account.name}
          </div>
          <div className="text-sm text-gray-600 mb-2">
            <span className="font-medium">Type:</span> {account.metaAccountType}
            {account.gasless && <span className="ml-2 text-green-600">⚡ Gasless</span>}
          </div>
          <div className="text-sm text-gray-600 mb-1">
            <span className="font-medium">USDC Balance:</span> {account.usdcBalance}
          </div>
          <div className="text-sm text-gray-600">
            <span className="font-medium">NRN Balance:</span> {account.nrnBalance}
          </div>
        </div>
      )}

      {/* Purchase Form */}
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            USDC Amount
          </label>
          <input
            type="number"
            value={usdcAmount}
            onChange={(e) => setUsdcAmount(e.target.value)}
            placeholder="Enter USDC amount"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            min="0"
            step="0.01"
          />
          {usdcAmount && (
            <div className="mt-1 text-sm text-gray-600">
              ≈ {estimatedNRN} NRN (Rate: 1 USDC = {conversionRate} NRN)
            </div>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            NRN Recipient Address (Optional)
          </label>
          <input
            type="text"
            value={nrnTargetAddress}
            onChange={(e) => setNrnTargetAddress(e.target.value)}
            placeholder={account?.address || 'Enter NRN address'}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Memo (Optional)
          </label>
          <input
            type="text"
            value={memo}
            onChange={(e) => setMemo(e.target.value)}
            placeholder="Transaction memo"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <button
          onClick={handlePurchase}
          disabled={!isValidAmount || isConverting}
          className="w-full bg-green-600 text-white py-2 px-4 rounded-md hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isConverting ? 'Processing...' : 'Purchase NRN Tokens'}
        </button>

        {!isValidAmount && usdcAmount && (
          <div className="text-sm text-red-600">
            {parseFloat(usdcAmount) <= 0 
              ? 'Please enter a valid amount' 
              : 'Insufficient USDC balance'
            }
          </div>
        )}
      </div>

      {/* Conversion History */}
      {conversionHistory.length > 0 && (
        <div className="mt-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-3">Recent Conversions</h3>
          <div className="space-y-2 max-h-40 overflow-y-auto">
            {conversionHistory.map((conversion) => (
              <div key={conversion.transactionId} className="bg-gray-50 rounded p-3 text-sm">
                <div className="flex justify-between items-center">
                  <span>{conversion.usdcAmount} USDC → {conversion.nrnAmount} NRN</span>
                  <span className={`px-2 py-1 rounded text-xs ${
                    conversion.status === 'confirmed' ? 'bg-green-100 text-green-800' :
                    conversion.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-red-100 text-red-800'
                  }`}>
                    {conversion.status}
                  </span>
                </div>
                <div className="text-gray-600 mt-1">
                  {conversion.timestamp.toLocaleDateString()}
                  {conversion.gasUsed === '0' && (
                    <span className="ml-2 text-green-600">⚡ Gasless</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default USDCToNRNPurchase;
