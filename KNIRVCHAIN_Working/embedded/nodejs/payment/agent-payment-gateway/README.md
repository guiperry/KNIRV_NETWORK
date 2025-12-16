# KNIRVCHAIN Payment Gateway Client Example

This is a client application demonstrating how to integrate with the KNIRVCHAIN payment gateway. It shows how to accept payments via Stripe and Coinbase Commerce and have KNIRVCHAIN tokens automatically disbursed to customers.

## Features

- Purchase KNIRVCHAIN tokens with credit/debit cards via Stripe
- Purchase KNIRVCHAIN tokens with cryptocurrencies via Coinbase Commerce
- Automatic token disbursement to customer wallets
- Responsive UI with Bootstrap

## Prerequisites

- Node.js and npm installed
- Stripe account with API keys
- Coinbase Commerce account with API keys
- Running KNIRVCHAIN node in root mode

## Installation

1. Clone the repository:
```bash
git clone https://github.com/your-username/KNIRVCHAIN-payment-gateway-client.git
cd KNIRVCHAIN-payment-gateway-client
```

2. Install dependencies:
```bash
npm install
```

3. Configure API keys:
   - Open `server.js`
   - Replace `'s k_test_your_secret_key'` with your Stripe secret key
   - Replace `'your_api_key'` with your Coinbase Commerce API key
   - Open `index.html`
   - Replace `'pk_test_your_publishable_key'` with your Stripe publishable key

4. Start the server:
```bash
npm start
```

5. Open your browser and navigate to `http://localhost:3000`

## Usage

1. Enter the amount of USD you want to spend
2. Enter your KNIRVCHAIN wallet address
3. Select your preferred payment method (Credit Card or Crypto)
4. Click "Purchase Tokens"
5. Complete the payment process
6. Tokens will be automatically sent to your wallet

## Integration with KNIRVCHAIN Node

This example client application communicates with payment gateways (Stripe and Coinbase Commerce) to process payments. The payment gateways then send webhook notifications to your KNIRVCHAIN node, which processes the payments and disburses tokens.

Make sure your KNIRVCHAIN node:
1. Is running in root mode (`--root` flag)
2. Has the payment processor enabled
3. Is configured with the correct webhook secrets
4. Is publicly accessible to receive webhook notifications

## Webhook Configuration

### Stripe
- Webhook URL: `https://your-node-domain:8090/webhooks/stripe`
- Events to listen for: `charge.succeeded`, `checkout.session.completed`

### Coinbase Commerce
- Webhook URL: `https://your-node-domain:8090/webhooks/coinbase`
- Events to listen for: `charge:confirmed`

## Security Considerations

- Never expose API keys in client-side code
- Always use HTTPS in production
- Validate all user inputs, especially wallet addresses
- Implement rate limiting to prevent abuse

## License

This example is provided under the MIT License.

## Additional Resources

For more information, refer to the [KNIRVCHAIN Payment Gateway Integration Guide](../../docs/payment_gateway_integration_guide.md).