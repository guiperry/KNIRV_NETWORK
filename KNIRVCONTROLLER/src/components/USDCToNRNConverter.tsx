/**
 * USDC to NRN Converter Component
 * Handles USDC to NRN conversion using Abstraxion wallet
 */

import React, { useState, useEffect } from 'react';
import { DollarSign, Coins, ArrowRight, CheckCircle, AlertCircle, Loader } from 'lucide-react';
import { useAbstraxionWallet, ConversionResult } from '../services/AbstraxionWalletService';
import { rxdbService } from '../services/RxDBService';

interface ConversionRequest {
  usdcAmount: string;
  nrnTargetAddress?: string;
  memo?: string;
}

export default function USDCToNRNConverter() {
  const { account, isConnected, isLoading, convertUSDCToNRN, getUSDCBalance } = useAbstraxionWallet();

  const [usdcAmount, setUsdcAmount] = useState('');
  const [conversionRate] = useState(0.1); // 1 USDC = 10 NRN
  const [estimatedNRN, setEstimatedNRN] = useState('0');
  const [isConverting, setIsConverting] = useState(false);
  const [conversionSuccess, setConversionSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [usdcBalance, setUsdcBalance] = useState('0');

  // Update estimated NRN when USDC amount changes
  useEffect(() => {
    if (usdcAmount && !isNaN(parseFloat(usdcAmount))) {
      const amount = parseFloat(usdcAmount);
      const nrn = amount * conversionRate * 10; // conversionRate * 10 for display
      setEstimatedNRN(nrn.toFixed(2));
    } else {
      setEstimatedNRN('0');
    }
  }, [usdcAmount, conversionRate]);

  // Load USDC balance on mount
  useEffect(() => {
    const loadBalance = async () => {
      if (isConnected) {
        try {
          const balance = await getUSDCBalance();
          setUsdcBalance(balance);
        } catch (error) {
          console.error('Failed to load USDC balance:', error);
        }
      }
    };

    loadBalance();
  }, [isConnected, getUSDCBalance]);

  // Handle conversion
  const handleConvert = async () => {
    if (!isConnected) {
      setError('Please connect your XION wallet first');
      return;
    }

    if (!usdcAmount || parseFloat(usdcAmount) <= 0) {
      setError('Please enter a valid USDC amount');
      return;
    }

    if (parseFloat(usdcAmount) > parseFloat(usdcBalance)) {
      setError('Insufficient USDC balance');
      return;
    }

    setIsConverting(true);
    setError(null);

    try {
      const request: ConversionRequest = {
        usdcAmount: usdcAmount,
        memo: 'USDC to NRN conversion via KNIRV Controller'
      };

      const result = await convertUSDCToNRN(request);

      // Store conversion in RxDB
      await storeConversionInDatabase(result);

      setConversionSuccess(true);

      // Reset form after success
      setTimeout(() => {
        setUsdcAmount('');
        setEstimatedNRN('0');
        setConversionSuccess(false);
        // Refresh balance
        getUSDCBalance().then(setUsdcBalance);
      }, 3000);

    } catch (error) {
      console.error('Conversion failed:', error);
      setError(error instanceof Error ? error.message : 'Conversion failed');
    } finally {
      setIsConverting(false);
    }
  };

  // Store conversion in RxDB
  const storeConversionInDatabase = async (result: ConversionResult) => {
    try {
      if (!rxdbService.isDatabaseInitialized()) {
        await rxdbService.initialize();
      }

      const db = rxdbService.getDatabase();

      // Store conversion record
      await db.conversions.insert({
        id: result.transactionId,
        type: 'conversion',
        walletId: account?.id || 'default',
        transactionId: result.transactionId,
        usdcAmount: result.usdcAmount,
        nrnAmount: result.nrnAmount,
        rate: conversionRate,
        timestamp: typeof result.timestamp === 'number' ? result.timestamp : result.timestamp.getTime(),
        status: 'completed' as const
      });

    } catch (error) {
      console.error('Failed to store conversion:', error);
      // Don't throw - conversion was successful, just failed to store locally
    }
  };

  const maxAmount = parseFloat(usdcBalance);
  const isValidAmount = usdcAmount && parseFloat(usdcAmount) > 0 && parseFloat(usdcAmount) <= maxAmount;

  return (
    <div className="bg-gray-800/80 backdrop-blur-xl rounded-xl p-6 border border-gray-600/50">
      <div className="flex items-center space-x-3 mb-6">
        <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-blue-500 rounded-lg flex items-center justify-center">
          <DollarSign className="w-5 h-5 text-white" />
        </div>
        <div>
          <h3 className="text-lg font-semibold text-white">USDC to NRN Converter</h3>
          <p className="text-sm text-gray-400">Convert USDC to NRN tokens</p>
        </div>
      </div>

      {/* Connection Status */}
      {!isConnected && (
        <div className="mb-4 p-3 bg-orange-500/20 border border-orange-500/30 rounded-lg">
          <div className="flex items-center space-x-2">
            <AlertCircle className="w-4 h-4 text-orange-400" />
            <span className="text-sm text-orange-400">Connect your XION wallet to convert</span>
          </div>
        </div>
      )}

      {/* Balance Display */}
      <div className="mb-6 grid grid-cols-2 gap-4">
        <div className="p-3 bg-gray-700/50 rounded-lg">
          <div className="text-xs text-gray-400 mb-1">USDC Balance</div>
          <div className="text-lg font-semibold text-white">{usdcBalance} USDC</div>
        </div>
        <div className="p-3 bg-gray-700/50 rounded-lg">
          <div className="text-xs text-gray-400 mb-1">Conversion Rate</div>
          <div className="text-lg font-semibold text-white">1 USDC = {conversionRate * 10} NRN</div>
        </div>
      </div>

      {/* Conversion Form */}
      <div className="space-y-4">
        {/* USDC Amount Input */}
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            USDC Amount
          </label>
          <div className="relative">
            <input
              type="number"
              value={usdcAmount}
              onChange={(e) => setUsdcAmount(e.target.value)}
              placeholder="0.00"
              className="w-full px-4 py-3 bg-gray-700/50 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:border-blue-400 focus:outline-none"
              step="0.01"
              min="0"
              max={maxAmount}
              disabled={!isConnected || isConverting || isLoading}
            />
            <div className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 text-sm">
              USDC
            </div>
          </div>
        </div>

        {/* Arrow */}
        <div className="flex justify-center">
          <ArrowRight className="w-5 h-5 text-gray-400" />
        </div>

        {/* Estimated NRN Output */}
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Estimated NRN Output
          </label>
          <div className="relative">
            <input
              type="text"
              value={estimatedNRN}
              readOnly
              className="w-full px-4 py-3 bg-gray-700/30 border border-gray-600 rounded-lg text-white"
            />
            <div className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 text-sm">
              NRN
            </div>
          </div>
        </div>

        {/* Error Display */}
        {error && (
          <div className="p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
            <div className="flex items-center space-x-2">
              <AlertCircle className="w-4 h-4 text-red-400" />
              <span className="text-sm text-red-400">{error}</span>
            </div>
          </div>
        )}

        {/* Success Display */}
        {conversionSuccess && (
          <div className="p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
            <div className="flex items-center space-x-2">
              <CheckCircle className="w-4 h-4 text-green-400" />
              <span className="text-sm text-green-400">
                Conversion successful! {estimatedNRN} NRN tokens are now available.
              </span>
            </div>
          </div>
        )}

        {/* Convert Button */}
        <button
          onClick={handleConvert}
          disabled={!isConnected || !isValidAmount || isConverting || isLoading}
          className="w-full py-3 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 disabled:from-gray-600 disabled:to-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-medium transition-all disabled:opacity-50"
        >
          {isConverting ? (
            <div className="flex items-center justify-center space-x-2">
              <Loader className="w-4 h-4 animate-spin" />
              <span>Converting...</span>
            </div>
          ) : (
            <div className="flex items-center justify-center space-x-2">
              <Coins className="w-4 h-4" />
              <span>Convert USDC to NRN</span>
            </div>
          )}
        </button>
      </div>

      {/* Info Text */}
      <div className="mt-6 p-3 bg-gray-700/30 rounded-lg">
        <p className="text-xs text-gray-400 text-center">
          Conversion rate is subject to market conditions. Actual rate may vary at time of transaction.
        </p>
      </div>
    </div>
  );
}
