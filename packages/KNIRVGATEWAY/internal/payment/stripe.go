package payment

import (
	"fmt"
	"math"
	"net/url"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
)

// StripeClient handles Stripe payment processing
type StripeClient struct {
	secretKey string
}

// NewStripeClient creates a new Stripe client
func NewStripeClient(secretKey string) *StripeClient {
	stripe.Key = secretKey
	return &StripeClient{
		secretKey: secretKey,
	}
}

// CreateCheckoutSession creates a Stripe checkout session
func (sc *StripeClient) CreateCheckoutSession(req *PaymentRequest, networkConfig *NetworkConfig) (*StripeCheckoutSession, error) {
	// Calculate token amount (assuming 1 USD = 10 tokens)
	tokenAmount := int64(math.Floor(req.Amount * 10))

	// Create checkout session parameters
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(string(stripe.CurrencyUSD)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(fmt.Sprintf("%s Tokens", networkConfig.Token)),
						Description: stripe.String(fmt.Sprintf("Purchase %d %s tokens on %s", tokenAmount, networkConfig.Token, networkConfig.Name)),
					},
					UnitAmount: stripe.Int64(int64(req.Amount * 100)), // Convert dollars to cents
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(fmt.Sprintf("http://localhost:8080/success.html?network=%s", url.QueryEscape(req.Network))),
		CancelURL:  stripe.String(fmt.Sprintf("http://localhost:8080/cancel.html?network=%s", url.QueryEscape(req.Network))),
		Metadata: map[string]string{
			"wallet_address": req.Address,
			"network":        req.Network,
			"token":          networkConfig.Token,
			"token_amount":   fmt.Sprintf("%d", tokenAmount),
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return &StripeCheckoutSession{
		ID:  sess.ID,
		URL: sess.URL,
	}, nil
}
