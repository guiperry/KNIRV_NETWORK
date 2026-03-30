'use client';

import React, { useState } from 'react';
import { X, CreditCard, Wallet, Coins, ArrowRight, CheckCircle, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { createStripeCheckoutSession, createPayPalOrder, capturePayPalOrder, type NRNPurchaseRequest } from '@/lib/api';

interface PaymentGatewayModalProps {
  isOpen: boolean;
  onClose: () => void;
  currentBalance?: number;
}

type PaymentMethod = 'stripe' | 'paypal';
type PurchaseStep = 'select' | 'processing' | 'success' | 'error';

const PACKAGES = [
  { nrn: 100, price: 10, popular: false },
  { nrn: 500, price: 45, popular: false },
  { nrn: 1000, price: 85, popular: true },
  { nrn: 5000, price: 400, popular: false },
  { nrn: 10000, price: 750, popular: false },
];

const PaymentGatewayModal: React.FC<PaymentGatewayModalProps> = ({ isOpen, onClose, currentBalance = 0 }) => {
  const [selectedPackage, setSelectedPackage] = useState<number | null>(null);
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('stripe');
  const [step, setStep] = useState<PurchaseStep>('select');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handlePurchase = async () => {
    if (selectedPackage === null) return;
    
    const pkg = PACKAGES[selectedPackage];
    setLoading(true);
    setError(null);
    setStep('processing');

    try {
      if (paymentMethod === 'stripe') {
        const result = await createStripeCheckoutSession({
          amount: pkg.price,
          currency: 'usd',
          nrn_amount: pkg.nrn,
          success_url: `${window.location.origin}/payment/success`,
          cancel_url: `${window.location.origin}/payment/cancel`,
        });
        
        if (result.url) {
          window.location.href = result.url;
        } else {
          throw new Error('No payment URL returned');
        }
      } else {
        const order = await createPayPalOrder({
          amount: pkg.price,
          currency: 'USD',
          nrn_amount: pkg.nrn,
        });
        
        if (order.order_id) {
          const capture = await capturePayPalOrder(order.order_id);
          if (capture.status === 'COMPLETED') {
            setStep('success');
          } else {
            throw new Error('Payment not completed');
          }
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Payment failed');
      setStep('error');
    } finally {
      setLoading(false);
    }
  };

  const resetAndClose = () => {
    setSelectedPackage(null);
    setPaymentMethod('stripe');
    setStep('select');
    setError(null);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center">
      <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={resetAndClose} />
      
      <div className="relative w-full max-w-lg bg-slate-900 border border-blue-600/50 rounded-2xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95">
        <div className="flex items-center justify-between p-4 border-b border-blue-600/30 bg-slate-800/50">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <Coins className="w-5 h-5 text-blue-400" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-white">Purchase NRN Tokens</h2>
              <p className="text-xs text-slate-400">Current Balance: {currentBalance.toLocaleString()} NRN</p>
            </div>
          </div>
          <button onClick={resetAndClose} className="text-slate-400 hover:text-white p-1">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6">
          {step === 'select' && (
            <>
              <div className="mb-6">
                <p className="text-sm text-slate-400 mb-3">Select Package</p>
                <div className="grid grid-cols-3 gap-3">
                  {PACKAGES.map((pkg, idx) => (
                    <button
                      key={idx}
                      onClick={() => setSelectedPackage(idx)}
                      className={`p-3 rounded-xl border-2 transition-all text-center ${
                        selectedPackage === idx
                          ? 'border-blue-500 bg-blue-600/20'
                          : 'border-slate-700 hover:border-slate-600 bg-slate-800/50'
                      }`}
                    >
                      <div className="text-xl font-black text-white">{pkg.nrn}</div>
                      <div className="text-xs text-slate-400">NRN</div>
                      {pkg.popular && (
                        <div className="mt-1 text-[10px] bg-blue-600 text-white px-1.5 py-0.5 rounded">POPULAR</div>
                      )}
                      <div className="text-sm font-bold text-green-400 mt-1">${pkg.price}</div>
                    </button>
                  ))}
                </div>
              </div>

              <div className="mb-6">
                <p className="text-sm text-slate-400 mb-3">Payment Method</p>
                <div className="flex gap-3">
                  <button
                    onClick={() => setPaymentMethod('stripe')}
                    className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-xl border-2 transition-all ${
                      paymentMethod === 'stripe'
                        ? 'border-blue-500 bg-blue-600/20'
                        : 'border-slate-700 hover:border-slate-600 bg-slate-800/50'
                    }`}
                  >
                    <CreditCard className="w-5 h-5 text-slate-300" />
                    <span className="font-medium text-white">Stripe</span>
                  </button>
                  <button
                    onClick={() => setPaymentMethod('paypal')}
                    className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-xl border-2 transition-all ${
                      paymentMethod === 'paypal'
                        ? 'border-blue-500 bg-blue-600/20'
                        : 'border-slate-700 hover:border-slate-600 bg-slate-800/50'
                    }`}
                  >
                    <Wallet className="w-5 h-5 text-slate-300" />
                    <span className="font-medium text-white">PayPal</span>
                  </button>
                </div>
              </div>

              {selectedPackage !== null && (
                <div className="p-4 bg-slate-800/50 rounded-xl border border-slate-700 mb-6">
                  <div className="flex justify-between text-sm">
                    <span className="text-slate-400">Package</span>
                    <span className="text-white font-medium">{PACKAGES[selectedPackage].nrn} NRN</span>
                  </div>
                  <div className="flex justify-between text-sm mt-2">
                    <span className="text-slate-400">Price</span>
                    <span className="text-white font-medium">${PACKAGES[selectedPackage].price} USD</span>
                  </div>
                  <div className="flex justify-between text-sm mt-2 pt-2 border-t border-slate-700">
                    <span className="text-slate-400">Rate</span>
                    <span className="text-green-400 font-medium">
                      ${(PACKAGES[selectedPackage].price / PACKAGES[selectedPackage].nrn * 100).toFixed(2)}/NRN
                    </span>
                  </div>
                </div>
              )}

              <Button
                onClick={handlePurchase}
                disabled={selectedPackage === null}
                className="w-full bg-blue-600 hover:bg-blue-700 text-white font-bold py-3 rounded-xl"
              >
                <ArrowRight className="w-4 h-4 mr-2" />
                Purchase {selectedPackage !== null ? PACKAGES[selectedPackage].nrn : 0} NRN
              </Button>
            </>
          )}

          {step === 'processing' && (
            <div className="text-center py-12">
              <Loader2 className="w-12 h-12 text-blue-500 animate-spin mx-auto mb-4" />
              <p className="text-lg font-medium text-white">Processing Payment...</p>
              <p className="text-sm text-slate-400 mt-2">Please wait while we process your payment</p>
            </div>
          )}

          {step === 'success' && (
            <div className="text-center py-12">
              <CheckCircle className="w-16 h-16 text-green-500 mx-auto mb-4" />
              <p className="text-xl font-bold text-white">Payment Successful!</p>
              <p className="text-sm text-slate-400 mt-2">
                {selectedPackage !== null ? PACKAGES[selectedPackage].nrn : 0} NRN has been added to your wallet
              </p>
              <Button
                onClick={resetAndClose}
                className="mt-6 bg-green-600 hover:bg-green-700 text-white font-bold py-3 rounded-xl"
              >
                Done
              </Button>
            </div>
          )}

          {step === 'error' && (
            <div className="text-center py-12">
              <div className="w-16 h-16 rounded-full bg-red-600/20 flex items-center justify-center mx-auto mb-4">
                <X className="w-8 h-8 text-red-500" />
              </div>
              <p className="text-xl font-bold text-white">Payment Failed</p>
              <p className="text-sm text-red-400 mt-2">{error}</p>
              <Button
                onClick={() => setStep('select')}
                className="mt-6 bg-slate-600 hover:bg-slate-700 text-white font-bold py-3 rounded-xl"
              >
                Try Again
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default PaymentGatewayModal;
