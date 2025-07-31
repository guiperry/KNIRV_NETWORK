

---

**Source**: KNIRVROOT/docs/protocols/Payment_Processor_Protocol.md

# KNIRVCHAIN Payment Processor

The KNIRVCHAIN Payment Processor is a component that enables root nodes to accept payments in fiat currencies (USD, EUR, etc.) and cryptocurrencies (ETH, BTC, etc.) and automatically disburse KNIRVCHAIN tokens to customers.

## Overview

The payment processor integrates with popular payment gateways like Stripe and Coinbase Commerce to process payments and automatically disburse tokens to customers' KNIRVCHAIN wallets. This enables roots to monetize their content and services using KNIRVCHAIN tokens without having to handle payment processing themselves.

## Features

- Accept payments in fiat currencies via Stripe
- Accept payments in cryptocurrencies via Coinbase Commerce
- Automatic token disbursement to customer wallets
- Configurable token pricing
- Webhook-based integration with payment gateways
- Secure transaction signing
- Idempotent payment processing to prevent duplicate disbursements

## Architecture

The payment processor consists of the following components:

1. **Webhook Server**: Listens for webhook events from payment gateways
2. **Payment Processor**: Processes payments and calculates token amounts
3. **Token Disbursement**: Signs and broadcasts transactions to disburse tokens
4. **Master Wallet**: Holds tokens for disbursement to customers

## Configuration

The payment processor can be configured via the `config.json` file or environment variables:

```json
{
  "is_root": true,
  "payment_processor": {
    "enabled": true,
    "node_rpc": "http://127.0.0.1:5000",
    "token_symbol": "NRN",
    "token_decimals": 6,
    "usd_per_token": 0.10,
    "eth_per_token": 0.00005,
    "webhook_port": 8090,
    "stripe_webhook_secret": "whsec_your_stripe_webhook_secret",
    "stripe_secret_key": "sk_your_stripe_secret_key",
    "coinbase_api_key": "your_coinbase_api_key",
    "coinbase_webhook_secret": "your_coinbase_webhook_secret"
  }
}
```

### Environment Variables

For security, it's recommended to set sensitive credentials using environment variables:

```bash
export STRIPE_SECRET_KEY="sk_your_stripe_secret_key"
export STRIPE_WEBHOOK_SECRET="whsec_your_stripe_webhook_secret"
export COINBASE_API_KEY="your_coinbase_api_key"
export COINBASE_WEBHOOK_SECRET="your_coinbase_webhook_secret"
```

## Usage

To enable the payment processor, start your KNIRVCHAIN node with the `--root` flag:

```bash
./KNIRVCHAIN --root
```

This will:
1. Enable root mode
2. Initialize the payment processor
3. Start the webhook server
4. Load the master wallet for token disbursement

## Integration

To integrate the payment processor with your application, you need to:

1. Configure your payment gateways (Stripe, Coinbase Commerce) to send webhook events to your KNIRVCHAIN node
2. Include the customer's KNIRVCHAIN wallet address in the payment metadata
3. Create a user interface for customers to purchase tokens

For detailed integration instructions, see the [Payment Gateway Integration Guide](payment_gateway_integration_guide.md).

## Example Client

An example client application is provided in the `examples/payment_gateway_client` directory. This example demonstrates how to integrate with the payment processor using Stripe and Coinbase Commerce.

## Security Considerations

- **API Keys**: Never expose API keys in client-side code
- **Webhook Verification**: Always verify webhook signatures
- **HTTPS**: Use HTTPS for all API endpoints
- **Input Validation**: Validate all user inputs, especially wallet addresses
- **Rate Limiting**: Implement rate limiting to prevent abuse
- **Idempotency**: Handle duplicate webhook events gracefully

## Testing

Unit tests are provided in `payment_processor_test.go`. To run the tests:

```bash
go test -v ./... -run TestPaymentProcessor
```

For testing with real payment gateways, use test mode:
- Stripe: Use test API keys and [test card numbers](https://stripe.com/docs/testing)
- Coinbase Commerce: Use the test mode in the dashboard

## Troubleshooting

Common issues and solutions:

1. **Webhook Not Received**
   - Check firewall settings
   - Verify webhook URL is correct
   - Ensure your node is publicly accessible

2. **Token Disbursement Failed**
   - Check master wallet balance
   - Verify recipient address format
   - Check node logs for specific errors

3. **Invalid Signature Error**
   - Verify webhook secret is correctly configured
   - Check for clock synchronization issues

## License

This component is part of the KNIRVCHAIN project and is subject to the same license terms.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
