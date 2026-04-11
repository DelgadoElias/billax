package mercadopago

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// Client handles HTTP communication with Mercado Pago API
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// newClient creates a default client pointing to the Mercado Pago production API
func newClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.mercadopago.com",
	}
}

// newClientWithBaseURL creates a client with a custom base URL (for testing)
func newClientWithBaseURL(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}
}

// --- Mercado Pago wire types (internal) ---

// mpPayer contains payer information for Mercado Pago
type mpPayer struct {
	Email string `json:"email"`
}

// mpCreatePaymentRequest is the body sent to POST /v1/payments
type mpCreatePaymentRequest struct {
	TransactionAmount float64            `json:"transaction_amount"`
	Description       string             `json:"description,omitempty"`
	Payer             mpPayer            `json:"payer"`
	ExternalReference string             `json:"external_reference,omitempty"`
	Metadata          map[string]string  `json:"metadata,omitempty"`
}

// mpPayment is the response from Mercado Pago payment endpoints
type mpPayment struct {
	ID                int64    `json:"id"`
	Status            string   `json:"status"`
	StatusDetail      string   `json:"status_detail"`
	TransactionAmount float64  `json:"transaction_amount"`
	PaymentMethodID   string   `json:"payment_method_id"`
	PaymentTypeID     string   `json:"payment_type_id"`
	Card              *mpCard  `json:"card,omitempty"`
}

// mpCard contains card information from Mercado Pago
type mpCard struct {
	LastFourDigits string `json:"last_four_digits"`
}

// mpRefundRequest is the body sent to POST /v1/payments/{id}/refunds
type mpRefundRequest struct {
	Amount float64 `json:"amount"`
}

// mpRefund is the response from Mercado Pago refund endpoint
type mpRefund struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// mpErrorResponse is the error response structure from Mercado Pago
type mpErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
	Status  int    `json:"status"`
}

// --- API Methods ---

// CreatePayment creates a payment via Mercado Pago API with idempotency and retry logic
func (c *Client) CreatePayment(ctx context.Context, req mpCreatePaymentRequest, accessToken, idempotencyKey string) (*mpPayment, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/payments", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Idempotency-Key", idempotencyKey)

	resp, err := c.doWithRetry(ctx, body, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, c.readErrorResponse(resp.Body)
	}

	var payment mpPayment
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return nil, fmt.Errorf("decoding payment response: %w", err)
	}

	return &payment, nil
}

// CreateRefund creates a refund via Mercado Pago API
func (c *Client) CreateRefund(ctx context.Context, paymentID string, amount float64, accessToken string) (*mpRefund, error) {
	req := mpRefundRequest{Amount: amount}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling refund request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/payments/"+paymentID+"/refunds", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating refund request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(ctx, body, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, c.readErrorResponse(resp.Body)
	}

	var refund mpRefund
	if err := json.NewDecoder(resp.Body).Decode(&refund); err != nil {
		return nil, fmt.Errorf("decoding refund response: %w", err)
	}

	return &refund, nil
}

// doWithRetry executes an HTTP request with exponential backoff retry on 429 (rate limit)
// Max 4 attempts (3 retries) with delays: 1s, 2s, 4s + random jitter (up to 50% of delay)
func (c *Client) doWithRetry(ctx context.Context, bodyBytes []byte, originalReq *http.Request) (*http.Response, error) {
	const (
		maxRetries = 3
		baseDelay  = time.Second
	)

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff and jitter
			delay := baseDelay * time.Duration(1<<uint(attempt-1)) // 1s, 2s, 4s
			jitter := time.Duration(rand.Int63n(int64(delay) / 2))

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay + jitter):
			}
		}

		// Reconstruct request for this attempt (body consumed on each Do)
		req, err := http.NewRequestWithContext(ctx, originalReq.Method, originalReq.URL.String(),
			bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("recreating request for retry: %w", err)
		}

		// Copy headers
		req.Header = originalReq.Header.Clone()

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue // Retry on transport error
		}

		if resp.StatusCode == http.StatusTooManyRequests { // 429
			resp.Body.Close()
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("mercadopago: max retries exceeded after %d attempts: %w", maxRetries+1, lastErr)
}

// readErrorResponse parses an error response from Mercado Pago
func (c *Client) readErrorResponse(body io.Reader) error {
	var mpErr mpErrorResponse
	if err := json.NewDecoder(body).Decode(&mpErr); err != nil {
		return fmt.Errorf("unreadable error response: %w", err)
	}
	return fmt.Errorf("mercadopago: %s (%s)", mpErr.Message, mpErr.Error)
}

// --- Helper functions ---

// centavosToUnits converts centavos (int64) to Mercado Pago's primary currency unit (float64)
// Example: 150 centavos → 1.50 ARS
func centavosToUnits(centavos int64) float64 {
	return float64(centavos) / 100.0
}

// unitsToCentavos converts Mercado Pago's primary currency unit back to centavos
// Example: 1.50 ARS → 150 centavos
func unitsToCentavos(units float64) int64 {
	return int64(math.Round(units * 100))
}
