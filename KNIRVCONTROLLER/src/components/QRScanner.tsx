import { useEffect, useRef, useState } from 'react';
import QrScanner from 'qr-scanner';
import { Camera, X, Flashlight, FlashlightOff, Wallet, Send, CheckCircle, AlertCircle, Loader } from 'lucide-react';
import { KNIRVWalletIntegration, TransactionRequest, WalletTransaction } from '../sensory-shell/KNIRVWalletIntegration';
import { KNIRVChainIntegration } from '../sensory-shell/KNIRVChainIntegration';

interface QRScannerProps {
  onScan: (result: string) => void;
  onClose: () => void;
  isOpen: boolean;
  walletIntegration?: KNIRVWalletIntegration;
  chainIntegration?: KNIRVChainIntegration;
}

interface PaymentRequest {
  type: 'payment' | 'skill_invocation' | 'wallet_connect';
  amount?: string;
  recipient?: string;
  skillId?: string;
  skillName?: string;
  nrnCost?: string;
  memo?: string;
  sessionId?: string;
  expires?: number;
}

interface PaymentState {
  step: 'scanning' | 'confirming' | 'processing' | 'success' | 'error';
  request?: PaymentRequest;
  transaction?: WalletTransaction;
  error?: string;
}

interface QRData {
  version: string;
  type: string;
  session_id: string;
  desktop_id: string;
  target_id?: string;
  expires_at: number;
  endpoint: string;
  public_key: string;
  capabilities?: string[];
  encrypted_payload?: string;
  signature: string;
}

export default function QRScanner({
  onScan,
  onClose,
  isOpen,
  walletIntegration,
  chainIntegration
}: QRScannerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [qrScanner, setQrScanner] = useState<QrScanner | null>(null);
  const [hasFlash, setHasFlash] = useState(false);
  const [flashEnabled, setFlashEnabled] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);

  // Payment workflow state
  const [paymentState, setPaymentState] = useState<PaymentState>({ step: 'scanning' });
  const [userBalance, setUserBalance] = useState<{ nrn: string; balance: string }>({ nrn: '0', balance: '0' });

  useEffect(() => {
    if (isOpen && videoRef.current) {
      initializeScanner();
      loadUserBalance();
    }

    return () => {
      if (qrScanner) {
        qrScanner.destroy();
      }
    };
  }, [isOpen]);

  // Load user balance when component opens
  const loadUserBalance = async () => {
    if (walletIntegration) {
      try {
        const currentAccount = walletIntegration.getCurrentAccount();
        if (currentAccount) {
          const balance = await walletIntegration.getAccountBalance(currentAccount.id);
          setUserBalance({
            nrn: balance.nrnBalance,
            balance: balance.balance
          });
        }
      } catch (error) {
        console.error('Failed to load user balance:', error);
      }
    }
  };

  const initializeScanner = async () => {
    if (!videoRef.current) return;

    try {
      setError(null);
      setScanning(true);

      const scanner = new QrScanner(
        videoRef.current,
        (result) => handleScanResult(result.data),
        {
          highlightScanRegion: true,
          highlightCodeOutline: true,
          preferredCamera: 'environment', // Use back camera on mobile
        }
      );

      // Check if device has flash
      const hasFlashSupport = await QrScanner.hasCamera();
      setHasFlash(hasFlashSupport);

      await scanner.start();
      setQrScanner(scanner);
      setScanning(false);
    } catch (err) {
      console.error('Failed to initialize QR scanner:', err);
      setError('Failed to access camera. Please check permissions.');
      setScanning(false);
    }
  };

  const handleScanResult = (data: string) => {
    try {
      // Try to parse as payment request first
      let paymentRequest: PaymentRequest | null = null;

      try {
        const parsed = JSON.parse(data);

        // Check if it's a payment request
        if (parsed.type === 'payment' || parsed.type === 'skill_invocation' || parsed.type === 'wallet_connect') {
          paymentRequest = parsed as PaymentRequest;
        }
        // Legacy QR code format
        else if (parsed.version && parsed.session_id && parsed.desktop_id) {
          // Check if QR code has expired
          if (parsed.expires_at && Date.now() / 1000 > parsed.expires_at) {
            setError('QR code has expired');
            return;
          }

          console.log('Valid QR code scanned:', parsed);
          onScan(data);
          return;
        }
      } catch (parseError) {
        // Not JSON, might be a simple payment URI
        if (data.startsWith('knirv:') || data.startsWith('nrn:')) {
          paymentRequest = parsePaymentURI(data);
        }
      }

      if (paymentRequest) {
        // Handle payment workflow
        handlePaymentRequest(paymentRequest);
      } else {
        // Fallback to original scan handler
        console.log('QR code scanned:', data);
        onScan(data);
      }

    } catch (error) {
      console.error('QR scan error:', error);
      setError(error instanceof Error ? error.message : 'Invalid QR code');
    }
  };

  // Parse payment URI (e.g., knirv:pay?amount=100&recipient=knirv1abc...)
  const parsePaymentURI = (uri: string): PaymentRequest => {
    const url = new URL(uri);
    const params = new URLSearchParams(url.search);

    return {
      type: 'payment',
      amount: params.get('amount') || undefined,
      recipient: params.get('recipient') || undefined,
      memo: params.get('memo') || undefined,
      skillId: params.get('skill') || undefined,
      nrnCost: params.get('nrn') || undefined
    };
  };

  // Handle payment request workflow
  const handlePaymentRequest = (request: PaymentRequest) => {
    console.log('Payment request detected:', request);

    // Validate request
    if (request.expires && Date.now() > request.expires) {
      setError('Payment request has expired');
      return;
    }

    // Set payment state for confirmation
    setPaymentState({
      step: 'confirming',
      request
    });
  };

  // Process the payment
  const processPayment = async () => {
    if (!paymentState.request || !walletIntegration) {
      setError('Payment request or wallet not available');
      return;
    }

    setPaymentState(prev => ({ ...prev, step: 'processing' }));

    try {
      const request = paymentState.request;
      let transactionId: string;

      if (request.type === 'skill_invocation' && request.skillId && request.nrnCost) {
        // Handle skill invocation payment
        transactionId = await walletIntegration.invokeSkill({
          skillId: request.skillId,
          skillName: request.skillName || request.skillId,
          nrnCost: request.nrnCost,
          parameters: {},
          expectedOutput: {},
          timeout: 30000
        });
      } else if (request.type === 'payment' && request.amount && request.recipient) {
        // Handle regular payment
        const transactionRequest: TransactionRequest = {
          from: walletIntegration.getCurrentAccount()?.address || '',
          to: request.recipient,
          amount: request.amount,
          memo: request.memo,
          nrnAmount: request.nrnCost
        };

        transactionId = await walletIntegration.createTransaction(transactionRequest);
      } else {
        throw new Error('Unsupported payment type');
      }

      // Monitor transaction status
      const transaction = await walletIntegration.checkTransactionStatus(transactionId);

      setPaymentState({
        step: 'success',
        request,
        transaction
      });

      // Refresh balance
      await loadUserBalance();

    } catch (error) {
      console.error('Payment processing failed:', error);
      setPaymentState({
        step: 'error',
        request: paymentState.request,
        error: error instanceof Error ? error.message : 'Payment failed'
      });
    }
  };

  // Cancel payment and return to scanning
  const cancelPayment = () => {
    setPaymentState({ step: 'scanning' });
    setError(null);
  };

  const toggleFlash = async () => {
    if (qrScanner && hasFlash) {
      try {
        // Note: setFlash method may not be available in all QrScanner versions
        // This is a simplified implementation
        setFlashEnabled(!flashEnabled);
        console.log('Flash toggle requested:', !flashEnabled);
      } catch (err) {
        console.error('Failed to toggle flash:', err);
      }
    }
  };

  const handleClose = () => {
    if (qrScanner) {
      qrScanner.destroy();
      setQrScanner(null);
    }
    setError(null);
    setFlashEnabled(false);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black z-50 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 bg-gray-900 text-white">
        <h2 className="text-lg font-semibold flex items-center gap-2">
          <Camera size={20} />
          Scan QR Code
        </h2>
        <div className="flex items-center gap-2">
          {hasFlash && (
            <button
              onClick={toggleFlash}
              className="p-2 rounded-full bg-gray-700 hover:bg-gray-600 transition-colors"
            >
              {flashEnabled ? <FlashlightOff size={20} /> : <Flashlight size={20} />}
            </button>
          )}
          <button
            onClick={handleClose}
            className="p-2 rounded-full bg-gray-700 hover:bg-gray-600 transition-colors"
          >
            <X size={20} />
          </button>
        </div>
      </div>

      {/* Scanner Area or Payment UI */}
      <div className="flex-1 relative">
        {paymentState.step === 'scanning' ? (
          <>
            <video
              ref={videoRef}
              className="w-full h-full object-cover"
              playsInline
              muted
            />

            {/* Scanning overlay */}
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="relative">
                {/* Scanning frame */}
                <div className="w-64 h-64 border-2 border-white rounded-lg relative">
                  <div className="absolute top-0 left-0 w-8 h-8 border-t-4 border-l-4 border-blue-500 rounded-tl-lg"></div>
                  <div className="absolute top-0 right-0 w-8 h-8 border-t-4 border-r-4 border-blue-500 rounded-tr-lg"></div>
                  <div className="absolute bottom-0 left-0 w-8 h-8 border-b-4 border-l-4 border-blue-500 rounded-bl-lg"></div>
                  <div className="absolute bottom-0 right-0 w-8 h-8 border-b-4 border-r-4 border-blue-500 rounded-br-lg"></div>

                  {scanning && (
                    <div className="absolute inset-0 flex items-center justify-center">
                      <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </>
        ) : (
          /* Payment Workflow UI */
          <div className="flex-1 bg-gray-900 p-6 flex flex-col justify-center">
            {paymentState.step === 'confirming' && paymentState.request && (
              <div className="max-w-md mx-auto w-full space-y-6">
                <div className="text-center">
                  <Wallet className="w-16 h-16 mx-auto mb-4 text-blue-400" />
                  <h3 className="text-xl font-semibold text-white mb-2">Confirm Payment</h3>
                  <p className="text-gray-400">Review the payment details below</p>
                </div>

                <div className="bg-gray-800 rounded-lg p-4 space-y-3">
                  <div className="flex justify-between">
                    <span className="text-gray-400">Type:</span>
                    <span className="text-white capitalize">{paymentState.request.type.replace('_', ' ')}</span>
                  </div>

                  {paymentState.request.skillName && (
                    <div className="flex justify-between">
                      <span className="text-gray-400">Skill:</span>
                      <span className="text-white">{paymentState.request.skillName}</span>
                    </div>
                  )}

                  {paymentState.request.amount && (
                    <div className="flex justify-between">
                      <span className="text-gray-400">Amount:</span>
                      <span className="text-white">{paymentState.request.amount} KNIRV</span>
                    </div>
                  )}

                  {paymentState.request.nrnCost && (
                    <div className="flex justify-between">
                      <span className="text-gray-400">NRN Cost:</span>
                      <span className="text-yellow-400">{paymentState.request.nrnCost} NRN</span>
                    </div>
                  )}

                  {paymentState.request.recipient && (
                    <div className="flex justify-between">
                      <span className="text-gray-400">To:</span>
                      <span className="text-white font-mono text-sm">{paymentState.request.recipient.slice(0, 20)}...</span>
                    </div>
                  )}
                </div>

                <div className="bg-gray-800 rounded-lg p-4">
                  <div className="flex justify-between mb-2">
                    <span className="text-gray-400">Your Balance:</span>
                    <span className="text-white">{userBalance.balance} KNIRV</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">NRN Balance:</span>
                    <span className="text-yellow-400">{userBalance.nrn} NRN</span>
                  </div>
                </div>

                <div className="flex space-x-3">
                  <button
                    onClick={cancelPayment}
                    className="flex-1 py-3 px-4 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={processPayment}
                    className="flex-1 py-3 px-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center justify-center space-x-2"
                  >
                    <Send className="w-4 h-4" />
                    <span>Confirm Payment</span>
                  </button>
                </div>
              </div>
            )}

            {paymentState.step === 'processing' && (
              <div className="max-w-md mx-auto w-full text-center space-y-6">
                <Loader className="w-16 h-16 mx-auto text-blue-400 animate-spin" />
                <div>
                  <h3 className="text-xl font-semibold text-white mb-2">Processing Payment</h3>
                  <p className="text-gray-400">Please wait while we process your transaction...</p>
                </div>
              </div>
            )}

            {paymentState.step === 'success' && paymentState.transaction && (
              <div className="max-w-md mx-auto w-full text-center space-y-6">
                <CheckCircle className="w-16 h-16 mx-auto text-green-400" />
                <div>
                  <h3 className="text-xl font-semibold text-white mb-2">Payment Successful</h3>
                  <p className="text-gray-400">Your transaction has been processed</p>
                </div>

                <div className="bg-gray-800 rounded-lg p-4 text-left">
                  <div className="flex justify-between mb-2">
                    <span className="text-gray-400">Transaction ID:</span>
                    <span className="text-white font-mono text-sm">{paymentState.transaction.id}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Status:</span>
                    <span className="text-green-400">{paymentState.transaction.status}</span>
                  </div>
                </div>

                <button
                  onClick={handleClose}
                  className="w-full py-3 px-4 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors"
                >
                  Done
                </button>
              </div>
            )}

            {paymentState.step === 'error' && (
              <div className="max-w-md mx-auto w-full text-center space-y-6">
                <AlertCircle className="w-16 h-16 mx-auto text-red-400" />
                <div>
                  <h3 className="text-xl font-semibold text-white mb-2">Payment Failed</h3>
                  <p className="text-gray-400">{paymentState.error || 'An error occurred while processing your payment'}</p>
                </div>

                <div className="flex space-x-3">
                  <button
                    onClick={cancelPayment}
                    className="flex-1 py-3 px-4 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors"
                  >
                    Back to Scanner
                  </button>
                  <button
                    onClick={processPayment}
                    className="flex-1 py-3 px-4 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors"
                  >
                    Retry
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Instructions */}
      <div className="p-4 bg-gray-900 text-white text-center">
        <p className="text-sm">
          Position the QR code within the frame to scan
        </p>
        {error && (
          <p className="text-red-400 text-sm mt-2">
            {error}
          </p>
        )}
      </div>
    </div>
  );
}
