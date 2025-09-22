const express = require('express');
const stripe = require('stripe')(process.env.STRIPE_SECRET_KEY || 'your_secret_key');
const { Client, resources } = require('coinbase-commerce-node');
const bodyParser = require('body-parser');
const cors = require('cors');
const path = require('path');

// Initialize Express app
const app = express();
app.use(cors());
app.use(bodyParser.json());
app.use(express.static(path.join(__dirname)));

// Network detection and configuration
const NETWORKS = {
  'mainnet': {
    name: 'Mainnet',
    token: 'NRN',
    faucetEnabled: false,
    paymentEnabled: true
  },
  'public-testnet': {
    name: 'Public Testnet',
    token: 'NRV',
    faucetEnabled: true,
    paymentEnabled: false,
    faucetAmount: 1000 // 1000 NRV tokens
  },
  'private-testnet': {
    name: 'Private Testnet',
    token: 'pNRV',
    faucetEnabled: true,
    paymentEnabled: false,
    faucetAmount: 5000 // 5000 pNRV tokens
  }
};

// Rate limiting for faucet (simple in-memory store)
const faucetRateLimit = new Map();
const FAUCET_COOLDOWN = 24 * 60 * 60 * 1000; // 24 hours in milliseconds

// Initialize Coinbase Commerce client
const coinbaseClient = Client.init('your_api_key');
const { Charge } = resources;

// Network detection middleware
app.use((req, res, next) => {
  const network = req.query.network || req.headers['x-knirv-network'] || 'disconnected';
  req.knirvNetwork = NETWORKS[network] || null;
  req.networkId = network;
  next();
});

// Serve the HTML page
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

// Network status endpoint
app.get('/api/network-status', (req, res) => {
  res.json({
    network: req.networkId,
    config: req.knirvNetwork,
    timestamp: Date.now()
  });
});

// Testnet faucet endpoint
app.post('/api/faucet/request', async (req, res) => {
  try {
    const { address, network } = req.body;
    const networkConfig = NETWORKS[network];

    // Validate network
    if (!networkConfig || !networkConfig.faucetEnabled) {
      return res.status(400).json({
        error: 'Faucet not available for this network',
        network: network
      });
    }

    // Validate address format
    if (!address || (!address.startsWith('knirv') && !address.startsWith('0x'))) {
      return res.status(400).json({ error: 'Invalid wallet address format' });
    }

    // Check rate limiting
    const rateLimitKey = `${network}-${address}`;
    const lastRequest = faucetRateLimit.get(rateLimitKey);
    const now = Date.now();

    if (lastRequest && (now - lastRequest) < FAUCET_COOLDOWN) {
      const timeLeft = FAUCET_COOLDOWN - (now - lastRequest);
      const hoursLeft = Math.ceil(timeLeft / (60 * 60 * 1000));

      return res.status(429).json({
        error: 'Rate limit exceeded',
        message: `Please wait ${hoursLeft} hours before requesting again`,
        timeLeft: timeLeft
      });
    }

    // Simulate token distribution (in a real implementation, this would interact with the blockchain)
    const transactionId = `faucet-${network}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

    // Update rate limiting
    faucetRateLimit.set(rateLimitKey, now);

    // Log the faucet request
    console.log(`Faucet request: ${networkConfig.faucetAmount} ${networkConfig.token} sent to ${address} on ${networkConfig.name}`);

    res.json({
      success: true,
      transaction: {
        id: transactionId,
        amount: networkConfig.faucetAmount,
        token: networkConfig.token,
        recipient: address,
        network: networkConfig.name,
        timestamp: now
      },
      message: `Successfully sent ${networkConfig.faucetAmount} ${networkConfig.token} to your wallet`
    });

  } catch (error) {
    console.error('Faucet error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Faucet status endpoint
app.get('/api/faucet/status/:address', (req, res) => {
  const { address } = req.params;
  const { network } = req.query;

  if (!network || !NETWORKS[network]) {
    return res.status(400).json({ error: 'Invalid network' });
  }

  const rateLimitKey = `${network}-${address}`;
  const lastRequest = faucetRateLimit.get(rateLimitKey);
  const now = Date.now();

  if (!lastRequest) {
    return res.json({ canRequest: true, timeLeft: 0 });
  }

  const timeLeft = FAUCET_COOLDOWN - (now - lastRequest);
  const canRequest = timeLeft <= 0;

  res.json({
    canRequest,
    timeLeft: Math.max(0, timeLeft),
    lastRequest,
    hoursLeft: canRequest ? 0 : Math.ceil(timeLeft / (60 * 60 * 1000))
  });
});

// Create Stripe checkout session (Mainnet only)
app.post('/api/create-checkout-session', async (req, res) => {
  try {
    const { amount, address, network } = req.body;
    const networkConfig = NETWORKS[network];

    // Only allow payments on mainnet
    if (!networkConfig || !networkConfig.paymentEnabled) {
      return res.status(400).json({
        error: 'Payments not available for this network',
        suggestion: 'Use the faucet for testnet tokens'
      });
    }

    // Validate address
    if (!address || (!address.startsWith('knirv') && !address.startsWith('0x'))) {
      return res.status(400).json({ error: 'Invalid wallet address' });
    }

    // Create Checkout Session
    const session = await stripe.checkout.sessions.create({
      payment_method_types: ['card'],
      line_items: [
        {
          price_data: {
            currency: 'usd',
            product_data: {
              name: `${networkConfig.token} Tokens`,
              description: `Purchase ${Math.floor(amount / 0.10)} ${networkConfig.token} tokens on ${networkConfig.name}`,
            },
            unit_amount: Math.round(amount * 100), // Convert dollars to cents
          },
          quantity: 1,
        },
      ],
      mode: 'payment',
      success_url: `${req.protocol}://${req.get('host')}/success.html?network=${network}`,
      cancel_url: `${req.protocol}://${req.get('host')}/cancel.html?network=${network}`,
      metadata: {
        wallet_address: address,
        network: network,
        token: networkConfig.token
      },
    });

    res.json({ id: session.id });
  } catch (error) {
    console.error('Stripe error:', error);
    res.status(500).json({ error: error.message });
  }
});

// Create Coinbase Commerce charge (Mainnet only)
app.post('/api/create-coinbase-charge', async (req, res) => {
  try {
    const { amount, address, network } = req.body;
    const networkConfig = NETWORKS[network];

    // Only allow payments on mainnet
    if (!networkConfig || !networkConfig.paymentEnabled) {
      return res.status(400).json({
        error: 'Payments not available for this network',
        suggestion: 'Use the faucet for testnet tokens'
      });
    }

    // Validate address
    if (!address || (!address.startsWith('knirv') && !address.startsWith('0x'))) {
      return res.status(400).json({ error: 'Invalid wallet address' });
    }

    // Create Coinbase Commerce charge
    const chargeData = {
      name: `${networkConfig.token} Tokens`,
      description: `Purchase ${Math.floor(amount / 0.10)} ${networkConfig.token} tokens on ${networkConfig.name}`,
      pricing_type: 'fixed_price',
      local_price: {
        amount: amount.toString(),
        currency: 'USD',
      },
      metadata: {
        wallet_address: address,
        network: network,
        token: networkConfig.token
      },
      redirect_url: `${req.protocol}://${req.get('host')}/success.html?network=${network}`,
      cancel_url: `${req.protocol}://${req.get('host')}/cancel.html?network=${network}`,
    };

    const charge = await Charge.create(chargeData);
    res.json(charge);
  } catch (error) {
    console.error('Coinbase error:', error);
    res.status(500).json({ error: error.message });
  }
});

// Health check endpoint
app.get('/health', (req, res) => {
  const statusData = {
    status: 'online',
    version: '2.0.0',
    uptime: process.uptime(),
    timestamp: new Date().toISOString(),
    service: 'KNIRV Payment Gateway & Testnet Faucet',
    networks: NETWORKS,
    features: {
      faucet: true,
      payments: true,
      networkDetection: true
    }
  };
  res.status(200).json(statusData);
});

// Success page
app.get('/success.html', (req, res) => {
  const network = req.query.network || 'mainnet';
  const networkConfig = NETWORKS[network] || NETWORKS.mainnet;

  res.send(`
    <!DOCTYPE html>
    <html>
    <head>
      <title>Transaction Successful - KNIRV ${networkConfig.name}</title>
      <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
      <style>
        body { padding-top: 50px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; }
        .card { border-radius: 15px; box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1); backdrop-filter: blur(10px); background: rgba(255, 255, 255, 0.95); }
        .network-badge { background: linear-gradient(45deg, #667eea, #764ba2); color: white; padding: 5px 15px; border-radius: 20px; font-size: 0.9rem; }
      </style>
    </head>
    <body>
      <div class="container">
        <div class="row justify-content-center">
          <div class="col-md-6">
            <div class="card">
              <div class="card-header bg-success text-white text-center">
                <h4 class="mb-0">Transaction Successful!</h4>
                <span class="network-badge mt-2 d-inline-block">${networkConfig.name}</span>
              </div>
              <div class="card-body">
                <div class="text-center mb-4">
                  <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" fill="currentColor" class="bi bi-check-circle-fill text-success" viewBox="0 0 16 16">
                    <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zm-3.97-3.03a.75.75 0 0 0-1.08.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05z"/>
                  </svg>
                </div>
                <p class="text-center">Your ${networkConfig.paymentEnabled ? 'payment has been processed' : 'faucet request has been completed'} successfully. Your ${networkConfig.token} tokens will be sent to your wallet shortly.</p>
                <div class="alert alert-info">
                  <strong>Network:</strong> ${networkConfig.name}<br>
                  <strong>Token:</strong> ${networkConfig.token}
                </div>
                <div class="d-grid gap-2 mt-4">
                  <a href="/" class="btn btn-primary">Return to Gateway</a>
                  <a href="${req.get('referer') || '/'}" class="btn btn-outline-secondary">Back to Previous Page</a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </body>
    </html>
  `);
});

// Cancel page
app.get('/cancel.html', (req, res) => {
  const network = req.query.network || 'mainnet';
  const networkConfig = NETWORKS[network] || NETWORKS.mainnet;

  res.send(`
    <!DOCTYPE html>
    <html>
    <head>
      <title>Transaction Cancelled - KNIRV ${networkConfig.name}</title>
      <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
      <style>
        body { padding-top: 50px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; }
        .card { border-radius: 15px; box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1); backdrop-filter: blur(10px); background: rgba(255, 255, 255, 0.95); }
        .network-badge { background: linear-gradient(45deg, #667eea, #764ba2); color: white; padding: 5px 15px; border-radius: 20px; font-size: 0.9rem; }
      </style>
    </head>
    <body>
      <div class="container">
        <div class="row justify-content-center">
          <div class="col-md-6">
            <div class="card">
              <div class="card-header bg-warning text-center">
                <h4 class="mb-0">Transaction Cancelled</h4>
                <span class="network-badge mt-2 d-inline-block">${networkConfig.name}</span>
              </div>
              <div class="card-body">
                <div class="text-center mb-4">
                  <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" fill="currentColor" class="bi bi-x-circle-fill text-warning" viewBox="0 0 16 16">
                    <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zM5.354 4.646a.5.5 0 1 0-.708.708L7.293 8l-2.647 2.646a.5.5 0 0 0 .708.708L8 8.707l2.646 2.647a.5.5 0 0 0 .708-.708L8.707 8l2.647-2.646a.5.5 0 0 0-.708-.708L8 7.293 5.354 4.646z"/>
                  </svg>
                </div>
                <p class="text-center">Your ${networkConfig.paymentEnabled ? 'payment' : 'faucet request'} has been cancelled. No charges were made.</p>
                <div class="alert alert-info">
                  <strong>Network:</strong> ${networkConfig.name}<br>
                  <strong>Token:</strong> ${networkConfig.token}
                </div>
                <div class="d-grid gap-2 mt-4">
                  <a href="/" class="btn btn-primary">Try Again</a>
                  <a href="${req.get('referer') || '/'}" class="btn btn-outline-secondary">Back to Previous Page</a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </body>
    </html>
  `);
});

// Start server
const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
  console.log(`Open http://localhost:${PORT} in your browser`);
});
