package mercadopago

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DelgadoElias/billax/internal/provider"
)

// newTestServer creates a test HTTP server with a custom handler and returns it with a provider
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Provider) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	p := newWithClient(newClientWithBaseURL(ts.URL))
	return ts, p
}

// --- Provider Interface Tests ---

func TestGetProviderName(t *testing.T) {
	p := New()
	if got := p.GetProviderName(); got != "mercadopago" {
		t.Errorf("GetProviderName() = %q, want %q", got, "mercadopago")
	}
}

func TestCapabilities(t *testing.T) {
	p := New()
	caps := p.Capabilities()
	if !caps.Plans {
		t.Errorf("Capabilities().Plans = false, want true")
	}
	if caps.PayPerUse {
		t.Errorf("Capabilities().PayPerUse = true, want false")
	}
}

func TestRequiredConfig(t *testing.T) {
	p := New()
	required := p.RequiredConfig()
	expectedKeys := map[string]bool{
		"access_token":   true,
		"webhook_secret": true,
	}

	if len(required) != len(expectedKeys) {
		t.Errorf("RequiredConfig() returned %d keys, want %d", len(required), len(expectedKeys))
	}

	for _, key := range required {
		if !expectedKeys[key] {
			t.Errorf("RequiredConfig() includes unexpected key %q", key)
		}
	}
}

// --- ValidateConfig Tests ---

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]string
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]string{
				"access_token":   "test_token",
				"webhook_secret": "test_secret",
			},
			wantErr: false,
		},
		{
			name: "missing access_token",
			config: map[string]string{
				"webhook_secret": "test_secret",
			},
			wantErr: true,
		},
		{
			name: "missing webhook_secret",
			config: map[string]string{
				"access_token": "test_token",
			},
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  map[string]string{},
			wantErr: true,
		},
		{
			name: "empty access_token",
			config: map[string]string{
				"access_token":   "",
				"webhook_secret": "test_secret",
			},
			wantErr: true,
		},
		{
			name: "empty webhook_secret",
			config: map[string]string{
				"access_token":   "test_token",
				"webhook_secret": "",
			},
			wantErr: true,
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- CreateCharge Tests ---

func TestCreateChargeMissingAccessToken(t *testing.T) {
	p := New()
	req := provider.ChargeRequest{
		Amount:   150,
		Currency: "ARS",
		Config: map[string]string{
			// Intentionally missing access_token
		},
	}

	result, err := p.CreateCharge(context.Background(), req)
	if result != nil {
		t.Errorf("CreateCharge() with missing token returned non-nil result: %v", result)
	}
	if err == nil {
		t.Error("CreateCharge() with missing token should return error")
	}
}

func TestCreateChargeHappyPath(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payments" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, "Bearer ") {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}

		// Verify idempotency key
		if r.Header.Get("X-Idempotency-Key") == "" {
			http.Error(w, "missing idempotency key", http.StatusBadRequest)
			return
		}

		// Read and verify request body
		var req mpCreatePaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		// Return a successful payment response
		resp := mpPayment{
			ID:              123,
			Status:          "approved",
			PaymentMethodID: "visa",
			PaymentTypeID:   "credit_card",
			Card: &mpCard{
				LastFourDigits: "4242",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	ts, p := newTestServer(t, handler)
	defer ts.Close()

	req := provider.ChargeRequest{
		Amount:             150,
		Currency:           "ARS",
		Description:        "Test charge",
		ExternalCustomerID: "user@example.com",
		IdempotencyKey:     "test-idempotency-key",
		Config: map[string]string{
			"access_token": "test_token",
		},
	}

	result, err := p.CreateCharge(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateCharge() error: %v", err)
	}

	if result.ProviderChargeID != "123" {
		t.Errorf("ProviderChargeID = %q, want %q", result.ProviderChargeID, "123")
	}
	if result.Status != provider.ChargeStatusSucceeded {
		t.Errorf("Status = %v, want %v", result.Status, provider.ChargeStatusSucceeded)
	}
	if result.PaymentMethod == nil || result.PaymentMethod.Brand != "visa" {
		t.Errorf("PaymentMethod.Brand = %v, want visa", result.PaymentMethod)
	}
	if result.PaymentMethod.LastFour != "4242" {
		t.Errorf("PaymentMethod.LastFour = %s, want 4242", result.PaymentMethod.LastFour)
	}
}

func TestCreateChargePending(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mpPayment{
			ID:     124,
			Status: "pending",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	ts, p := newTestServer(t, handler)
	defer ts.Close()

	result, err := p.CreateCharge(context.Background(), provider.ChargeRequest{
		Amount: 100,
		Config: map[string]string{"access_token": "token"},
		IdempotencyKey: "key",
	})
	if err != nil {
		t.Fatalf("CreateCharge() error: %v", err)
	}

	if result.Status != provider.ChargeStatusPending {
		t.Errorf("Status = %v, want %v", result.Status, provider.ChargeStatusPending)
	}
}

func TestCreateChargeRejected(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mpPayment{
			ID:           125,
			Status:       "rejected",
			StatusDetail: "cc_rejected_insufficient_amount",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	ts, p := newTestServer(t, handler)
	defer ts.Close()

	result, err := p.CreateCharge(context.Background(), provider.ChargeRequest{
		Amount: 100,
		Config: map[string]string{"access_token": "token"},
		IdempotencyKey: "key",
	})
	if err != nil {
		t.Fatalf("CreateCharge() error: %v", err)
	}

	if result.Status != provider.ChargeStatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, provider.ChargeStatusFailed)
	}
	if result.FailureReason != "cc_rejected_insufficient_amount" {
		t.Errorf("FailureReason = %s, want cc_rejected_insufficient_amount", result.FailureReason)
	}
}

func TestCreateChargeAmountConversion(t *testing.T) {
	var capturedAmount float64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mpCreatePaymentRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedAmount = req.TransactionAmount

		resp := mpPayment{ID: 126, Status: "approved"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	ts, p := newTestServer(t, handler)
	defer ts.Close()

	// 150 centavos = 1.50 in primary currency
	_, err := p.CreateCharge(context.Background(), provider.ChargeRequest{
		Amount:         150,
		Config:         map[string]string{"access_token": "token"},
		IdempotencyKey: "key",
	})
	if err != nil {
		t.Fatalf("CreateCharge() error: %v", err)
	}

	expected := 1.50
	if capturedAmount != expected {
		t.Errorf("Amount sent to MP = %f, want %f", capturedAmount, expected)
	}
}

func TestCreateChargeServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errResp := mpErrorResponse{
			Message: "internal error",
			Error:   "internal_error",
			Status:  500,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
	})

	ts, p := newTestServer(t, handler)
	defer ts.Close()

	_, err := p.CreateCharge(context.Background(), provider.ChargeRequest{
		Amount:         100,
		Config:         map[string]string{"access_token": "token"},
		IdempotencyKey: "key",
	})
	if err == nil {
		t.Error("CreateCharge() should return error on 500")
	}
	if !strings.Contains(err.Error(), "provider error") {
		t.Errorf("error should wrap provider error, got: %v", err)
	}
}

func TestCreateChargeRetryOn429(t *testing.T) {
	attempt := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// Third attempt succeeds
		resp := mpPayment{ID: 127, Status: "approved"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	ts, p := newTestServer(t, handler)
	defer ts.Close()

	result, err := p.CreateCharge(context.Background(), provider.ChargeRequest{
		Amount:         100,
		Config:         map[string]string{"access_token": "token"},
		IdempotencyKey: "key",
	})
	if err != nil {
		t.Fatalf("CreateCharge() error: %v", err)
	}
	if result.ProviderChargeID != "127" {
		t.Errorf("ProviderChargeID = %s, want 127 (should have retried)", result.ProviderChargeID)
	}
	if attempt < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", attempt)
	}
}

func TestCreateChargeRetryExhausted(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return 429
		w.WriteHeader(http.StatusTooManyRequests)
	})

	ts, p := newTestServer(t, handler)
	defer ts.Close()

	_, err := p.CreateCharge(context.Background(), provider.ChargeRequest{
		Amount:         100,
		Config:         map[string]string{"access_token": "token"},
		IdempotencyKey: "key",
	})
	if err == nil {
		t.Error("CreateCharge() should return error after exhausting retries")
	}
}

// --- RefundCharge Tests ---

func TestRefundChargeNotSupported(t *testing.T) {
	p := New()
	result, err := p.RefundCharge(context.Background(), "123", 100)
	if result != nil {
		t.Errorf("RefundCharge() should return nil result, got %v", result)
	}
	if err == nil {
		t.Error("RefundCharge() should return error (not supported)")
	}
}

// --- HandleWebhook Tests ---

func TestHandleWebhookValidPayment(t *testing.T) {
	p := New()
	secret := "my_webhook_secret"
	requestID := "req-123"
	ts := "1234567890"

	// Build a valid webhook payload
	payload := mpWebhookPayload{
		ID:   456,
		Type: "payment",
		Data: struct {
			ID string `json:"id"`
		}{ID: "456"},
	}
	payloadBytes, _ := json.Marshal(payload)

	// Compute a valid HMAC signature
	message := fmt.Sprintf("id:456;request-id:%s;ts:%s;", requestID, ts)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	v1 := hex.EncodeToString(mac.Sum(nil))
	rawSignature := fmt.Sprintf("ts=%s,v1=%s", ts, v1)

	// Encode signature in the required format
	signature := fmt.Sprintf("%s|%s|%s", secret, requestID, rawSignature)

	event, err := p.HandleWebhook(context.Background(), payloadBytes, signature)
	if err != nil {
		t.Fatalf("HandleWebhook() error: %v", err)
	}

	if event.ProviderChargeID != "456" {
		t.Errorf("ProviderChargeID = %s, want 456", event.ProviderChargeID)
	}
	if event.EventType != "payment" {
		t.Errorf("EventType = %s, want payment", event.EventType)
	}
}

func TestHandleWebhookInvalidSignature(t *testing.T) {
	p := New()
	secret := "my_webhook_secret"
	requestID := "req-123"
	ts := "1234567890"

	payload := mpWebhookPayload{
		ID:   457,
		Type: "payment",
		Data: struct {
			ID string `json:"id"`
		}{ID: "457"},
	}
	payloadBytes, _ := json.Marshal(payload)

	// Use wrong HMAC value
	wrongV1 := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	rawSignature := fmt.Sprintf("ts=%s,v1=%s", ts, wrongV1)
	signature := fmt.Sprintf("%s|%s|%s", secret, requestID, rawSignature)

	_, err := p.HandleWebhook(context.Background(), payloadBytes, signature)
	if err == nil {
		t.Error("HandleWebhook() should return error for invalid signature")
	}
}

func TestHandleWebhookMalformedSignatureParameter(t *testing.T) {
	p := New()
	payload := []byte(`{"id":458,"type":"payment"}`)

	// Only 2 parts instead of 3
	signature := "secret|requestID"

	_, err := p.HandleWebhook(context.Background(), payload, signature)
	if err == nil {
		t.Error("HandleWebhook() should return error for malformed signature parameter")
	}
}

func TestHandleWebhookNonPaymentEvent(t *testing.T) {
	p := New()
	secret := "my_webhook_secret"
	requestID := "req-123"
	ts := "1234567890"

	payload := mpWebhookPayload{
		ID:   459,
		Type: "plan",
		Data: struct {
			ID string `json:"id"`
		}{ID: "plan-123"},
	}
	payloadBytes, _ := json.Marshal(payload)

	// Compute valid signature
	message := fmt.Sprintf("id:plan-123;request-id:%s;ts:%s;", requestID, ts)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	v1 := hex.EncodeToString(mac.Sum(nil))
	rawSignature := fmt.Sprintf("ts=%s,v1=%s", ts, v1)
	signature := fmt.Sprintf("%s|%s|%s", secret, requestID, rawSignature)

	event, err := p.HandleWebhook(context.Background(), payloadBytes, signature)
	if err != nil {
		t.Fatalf("HandleWebhook() error: %v", err)
	}

	if event.EventType != "plan" {
		t.Errorf("EventType = %s, want plan", event.EventType)
	}
	// Non-payment events should not have ProviderChargeID set (or empty)
}

func TestHandleWebhookMalformedJSON(t *testing.T) {
	p := New()
	payload := []byte(`{broken json}`)
	signature := "secret|requestID|ts=123,v1=abc"

	_, err := p.HandleWebhook(context.Background(), payload, signature)
	if err == nil {
		t.Error("HandleWebhook() should return error for malformed JSON")
	}
}

// --- Conversion Helper Tests ---

func TestCentavosToUnits(t *testing.T) {
	tests := []struct {
		centavos int64
		want     float64
	}{
		{100, 1.0},
		{150, 1.5},
		{1, 0.01},
		{0, 0.0},
		{9999, 99.99},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d centavos", tt.centavos), func(t *testing.T) {
			got := centavosToUnits(tt.centavos)
			if got != tt.want {
				t.Errorf("centavosToUnits(%d) = %f, want %f", tt.centavos, got, tt.want)
			}
		})
	}
}

func TestUnitsToCentavos(t *testing.T) {
	tests := []struct {
		units float64
		want  int64
	}{
		{1.0, 100},
		{1.5, 150},
		{0.01, 1},
		{0.0, 0},
		{99.99, 9999},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.2f units", tt.units), func(t *testing.T) {
			got := unitsToCentavos(tt.units)
			if got != tt.want {
				t.Errorf("unitsToCentavos(%.2f) = %d, want %d", tt.units, got, tt.want)
			}
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	testCases := []int64{100, 150, 1, 9999, 50000}
	for _, centavos := range testCases {
		t.Run(fmt.Sprintf("roundtrip %d", centavos), func(t *testing.T) {
			units := centavosToUnits(centavos)
			backToCentavos := unitsToCentavos(units)
			if backToCentavos != centavos {
				t.Errorf("roundtrip(%d) = %d, want %d", centavos, backToCentavos, centavos)
			}
		})
	}
}

// --- Helper for manual testing signature generation ---
// This can be useful for debugging webhook signature validation
func createTestWebhookSignature(secret, requestID, paymentID, ts string) string {
	message := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", paymentID, requestID, ts)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	v1 := hex.EncodeToString(mac.Sum(nil))
	rawSignature := fmt.Sprintf("ts=%s,v1=%s", ts, v1)
	return fmt.Sprintf("%s|%s|%s", secret, requestID, rawSignature)
}

func TestCreateTestWebhookSignature(t *testing.T) {
	// Verify the helper works
	sig := createTestWebhookSignature("secret", "req-1", "pay-1", "123")
	if !strings.Contains(sig, "secret") || !strings.Contains(sig, "req-1") {
		t.Errorf("createTestWebhookSignature produced unexpected result: %s", sig)
	}
}
