import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './payment-gateway.module.css';

export default function PaymentGateway() {
  const { activePage } = useNavigation('payment-gateway');
  const [amount, setAmount] = useState(10);
  const [walletAddress, setWalletAddress] = useState('');
  const [paymentMethod, setPaymentMethod] = useState('stripe');
  const [isProcessing, setIsProcessing] = useState(false);
  const [error, setError] = useState('');

  const tokenPrice = 0.10; // $0.10 per token
  const tokenAmount = Math.floor(amount / tokenPrice);

  const handlePayment = async () => {
    if (!walletAddress.startsWith('KNIRVORACLE')) {
      setError('Please enter a valid KNIRVORACLE wallet address');
      return;
    }

    setIsProcessing(true);
    setError('');

    try {
      const requestBody = {
        amount: amount,
        KNIRVORACLE_address: walletAddress
      };

      if (paymentMethod === 'stripe') {
        const response = await fetch('/api/create-checkout-session', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(requestBody),
        });

        const session = await response.json();

        if (!response.ok) {
          throw new Error(session.error || 'Failed to create checkout session');
        }

        // Redirect to Stripe Checkout
        const stripe = window.Stripe('pk_test_your_publishable_key'); // Replace with actual key
        const result = await stripe.redirectToCheckout({
          sessionId: session.id
        });

        if (result.error) {
          throw new Error(result.error.message);
        }
      } else if (paymentMethod === 'coinbase') {
        const response = await fetch('/api/create-coinbase-charge', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(requestBody),
        });

        const charge = await response.json();

        if (!response.ok) {
          throw new Error(charge.error || 'Failed to create Coinbase charge');
        }

        // Redirect to Coinbase Commerce checkout
        window.location.href = charge.hosted_url;
      }
    } catch (error) {
      console.error('Payment error:', error);
      setError('Payment processing error: ' + error.message);
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Payment Gateway">
      <PageHeader
        title="KNIRVORACLE Token Purchase"
        subtitle="Purchase KNIRVORACLE tokens securely using credit card or cryptocurrency"
        titleColor="#007bff"
      />

      <div className={styles.container}>
        <GlassyCard className={styles.purchaseCard}>
          <div className={styles.cardHeader}>
            <h4>Purchase KNIRVORACLE Tokens</h4>
          </div>
          <div className={styles.cardBody}>
            {/* Token Info Card */}
            <GlassyCard className={styles.tokenCard}>
              <div className={styles.tokenInfo}>
                <div>
                  <h5>KNIRVORACLE Token (NRN)</h5>
                  <p>Current price: ${tokenPrice.toFixed(2)} per token</p>
                </div>
                <div className={styles.tokenAmounts}>
                  <h5>${amount.toFixed(2)}</h5>
                  <p>{tokenAmount} NRN</p>
                </div>
              </div>
            </GlassyCard>

            {/* Purchase Form */}
            <div className={styles.purchaseForm}>
              <div className={styles.formGroup}>
                <label htmlFor="amount">Amount (USD)</label>
                <div className={styles.inputGroup}>
                  <span className={styles.inputPrefix}>$</span>
                  <input
                    type="number"
                    id="amount"
                    value={amount}
                    onChange={(e) => setAmount(parseFloat(e.target.value) || 0)}
                    min="1"
                    step="1"
                    className={styles.formControl}
                  />
                </div>
                <div className={styles.formText}>Minimum purchase: $1.00</div>
              </div>

              <div className={styles.formGroup}>
                <label htmlFor="wallet-address">KNIRVORACLE Wallet Address</label>
                <input
                  type="text"
                  id="wallet-address"
                  value={walletAddress}
                  onChange={(e) => setWalletAddress(e.target.value)}
                  placeholder="KNIRVORACLE..."
                  className={styles.formControl}
                  required
                />
                <div className={styles.formText}>Enter the wallet address where you want to receive tokens</div>
              </div>

              <div className={styles.formGroup}>
                <label>Payment Method</label>
                <div className={styles.paymentOptions}>
                  <div
                    className={`${styles.paymentOption} ${paymentMethod === 'stripe' ? styles.selected : ''}`}
                    onClick={() => setPaymentMethod('stripe')}
                  >
                    <img src="https://upload.wikimedia.org/wikipedia/commons/b/ba/Stripe_Logo%2C_revised_2016.svg" alt="Stripe" />
                    <div>Credit Card</div>
                  </div>
                  <div
                    className={`${styles.paymentOption} ${paymentMethod === 'coinbase' ? styles.selected : ''}`}
                    onClick={() => setPaymentMethod('coinbase')}
                  >
                    <img src="https://seeklogo.com/images/C/coinbase-coin-logo-C86F46D7B8-seeklogo.com.png" alt="Coinbase" />
                    <div>Crypto</div>
                  </div>
                </div>
              </div>

              {error && (
                <div className={styles.errorAlert}>
                  {error}
                </div>
              )}

              <div className={styles.buttonGroup}>
                <button
                  type="button"
                  onClick={handlePayment}
                  disabled={isProcessing || !walletAddress}
                  className={styles.purchaseButton}
                >
                  {isProcessing ? (
                    <>
                      <span className={styles.spinner}></span>
                      Processing...
                    </>
                  ) : (
                    'Purchase Tokens'
                  )}
                </button>
              </div>
            </div>
          </div>
        </GlassyCard>

        {/* How It Works */}
        <GlassyCard className={styles.infoCard}>
          <div className={styles.cardHeader}>
            <h5>How It Works</h5>
          </div>
          <div className={styles.cardBody}>
            <ol className={styles.instructions}>
              <li>Enter the amount you want to purchase</li>
              <li>Enter your KNIRVORACLE wallet address</li>
              <li>Select your preferred payment method</li>
              <li>Complete the payment</li>
              <li>Tokens will be sent to your wallet automatically</li>
            </ol>
          </div>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}