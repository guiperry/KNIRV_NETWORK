const express = require('express');
const stripe = require('stripe')('sk_test_your_secret_key');
const { Client, resources } = require('coinbase-commerce-node');
const bodyParser = require('body-parser');
const path = require('path');

// Initialize Express app
const app = express();
app.use(bodyParser.json());
app.use(express.static(path.join(__dirname)));

// Initialize Coinbase Commerce client
const coinbaseClient = Client.init('your_api_key');
const { Charge } = resources;

// Serve the HTML page
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

// Create Stripe checkout session
app.post('/api/create-checkout-session', async (req, res) => {
  try {
    const { amount, KNIRVORACLE_address } = req.body;
    
    // Validate KNIRVORACLE address
    if (!KNIRVORACLE_address || !KNIRVORACLE_address.startsWith('KNIRVORACLE')) {
      return res.status(400).json({ error: 'Invalid KNIRVORACLE address' });
    }
    
    // Create Checkout Session
    const session = await stripe.checkout.sessions.create({
      payment_method_types: ['card'],
      line_items: [
        {
          price_data: {
            currency: 'usd',
            product_data: {
              name: 'KNIRVORACLE Tokens',
              description: `Purchase ${Math.floor(amount / 0.10)} KNIRVORACLE tokens`,
            },
            unit_amount: Math.round(amount * 100), // Convert dollars to cents
          },
          quantity: 1,
        },
      ],
      mode: 'payment',
      success_url: `${req.protocol}://${req.get('host')}/success.html`,
      cancel_url: `${req.protocol}://${req.get('host')}/cancel.html`,
      metadata: {
        KNIRVORACLE_address: KNIRVORACLE_address, // CRITICAL: Include the recipient address
      },
    });
    
    res.json({ id: session.id });
  } catch (error) {
    console.error('Stripe error:', error);
    res.status(500).json({ error: error.message });
  }
});

// Create Coinbase Commerce charge
app.post('/api/create-coinbase-charge', async (req, res) => {
  try {
    const { amount, KNIRVORACLE_address } = req.body;
    
    // Validate KNIRVORACLE address
    if (!KNIRVORACLE_address || !KNIRVORACLE_address.startsWith('KNIRVORACLE')) {
      return res.status(400).json({ error: 'Invalid KNIRVORACLE address' });
    }
    
    // Create Coinbase Commerce charge
    const chargeData = {
      name: 'KNIRVORACLE Tokens',
      description: `Purchase ${Math.floor(amount / 0.10)} KNIRVORACLE tokens`,
      pricing_type: 'fixed_price',
      local_price: {
        amount: amount.toString(),
        currency: 'USD',
      },
      metadata: {
        KNIRVORACLE_address: KNIRVORACLE_address, // CRITICAL: Include the recipient address
      },
      redirect_url: `${req.protocol}://${req.get('host')}/success.html`,
      cancel_url: `${req.protocol}://${req.get('host')}/cancel.html`,
    };
    
    const charge = await Charge.create(chargeData);
    res.json(charge);
  } catch (error) {
    console.error('Coinbase error:', error);
    res.status(500).json({ error: error.message });
  }
});

// Success page
app.get('/success.html', (req, res) => {
  res.send(`
    <!DOCTYPE html>
    <html>
    <head>
      <title>Payment Successful</title>
      <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
      <style>
        body { padding-top: 50px; background-color: #f8f9fa; }
        .card { border-radius: 10px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
      </style>
    </head>
    <body>
      <div class="container">
        <div class="row justify-content-center">
          <div class="col-md-6">
            <div class="card">
              <div class="card-header bg-success text-white">
                <h4 class="mb-0">Payment Successful!</h4>
              </div>
              <div class="card-body">
                <div class="text-center mb-4">
                  <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" fill="currentColor" class="bi bi-check-circle-fill text-success" viewBox="0 0 16 16">
                    <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zm-3.97-3.03a.75.75 0 0 0-1.08.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05z"/>
                  </svg>
                </div>
                <p class="text-center">Your payment has been processed successfully. Your KNIRVORACLE tokens will be sent to your wallet shortly.</p>
                <div class="d-grid gap-2 mt-4">
                  <a href="/" class="btn btn-primary">Return to Home</a>
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
  res.send(`
    <!DOCTYPE html>
    <html>
    <head>
      <title>Payment Cancelled</title>
      <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
      <style>
        body { padding-top: 50px; background-color: #f8f9fa; }
        .card { border-radius: 10px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
      </style>
    </head>
    <body>
      <div class="container">
        <div class="row justify-content-center">
          <div class="col-md-6">
            <div class="card">
              <div class="card-header bg-warning">
                <h4 class="mb-0">Payment Cancelled</h4>
              </div>
              <div class="card-body">
                <p class="text-center">Your payment has been cancelled. No charges were made.</p>
                <div class="d-grid gap-2 mt-4">
                  <a href="/" class="btn btn-primary">Try Again</a>
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