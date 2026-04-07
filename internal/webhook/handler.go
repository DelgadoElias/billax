package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/payment"
	"github.com/DelgadoElias/billax/internal/provider"
)

// Handler handles incoming webhooks from payment providers
type Handler struct {
	paymentRepo   payment.PaymentRepo
	credentialsDB CredentialsProvider // interface to get webhook secret for provider
	adapter       *provider.ProviderAdapter
}

// CredentialsProvider defines how to fetch provider credentials
type CredentialsProvider interface {
	// GetCredentials fetches credentials for a provider given tenant context
	// In practice, this is injected from providercredentials service
	GetCredentials(tenantID uuid.UUID, providerName string) (map[string]string, error)
}

// NewHandler creates a new webhook handler
func NewHandler(paymentRepo payment.PaymentRepo, credentialsDB CredentialsProvider, adapter *provider.ProviderAdapter) *Handler {
	return &Handler{
		paymentRepo:   paymentRepo,
		credentialsDB: credentialsDB,
		adapter:       adapter,
	}
}

// HandleMercadoPagoWebhook handles POST /webhooks/mercadopago
// Expected headers:
//   - x-signature: Mercado Pago HMAC signature
//   - x-request-id: Request ID for signature validation
// Expected query params:
//   - data.id: Payment ID (sent by MP in the body's data.id field)
func (h *Handler) HandleMercadoPagoWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Extract headers
	xSignature := r.Header.Get("X-Signature")
	xRequestID := r.Header.Get("X-Request-ID")

	if xSignature == "" || xRequestID == "" {
		http.Error(w, "missing X-Signature or X-Request-ID header", http.StatusBadRequest)
		return
	}

	// Parse webhook body to extract payment ID
	var webhookBody struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &webhookBody); err != nil {
		http.Error(w, "cannot parse webhook body", http.StatusBadRequest)
		return
	}

	// Extract tenant ID from query parameter or infer from payment lookup
	// For MVP: tenants send X-Tenant-ID header (in prod, derive from webhook secret)
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		http.Error(w, "missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Fetch webhook secret from credentials (this would use credentialsDB in real scenario)
	// For now, just validate the signature
	credentials, err := h.credentialsDB.GetCredentials(tenantID, "mercadopago")
	if err != nil {
		// Log the error but don't expose to MP
		http.Error(w, "cannot verify signature", http.StatusUnauthorized)
		return
	}

	webhookSecret := credentials["webhook_secret"]

	// Assemble signature in the convention expected by the connector:
	// "<webhook_secret>|<x_request_id>|<raw_x_signature>"
	signature := fmt.Sprintf("%s|%s|%s", webhookSecret, xRequestID, xSignature)

	// Validate and parse webhook via adapter
	event, err := h.adapter.HandleWebhook(r.Context(), "mercadopago", body, signature)
	if err != nil {
		// Signature validation failed
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// At this point, signature is valid
	// event.ProviderChargeID contains the payment ID
	// event.Status is ChargeStatusPending (MP webhooks don't include actual status)

	// For MVP: just acknowledge receipt
	// In production: would fetch actual payment status from MP and update our DB
	//
	// TODO: Implement in next phase
	// 1. Fetch payment from MP: GET /v1/payments/{event.ProviderChargeID}
	// 2. Update our payment record with actual status
	// 3. Trigger subscription renewal job if status is succeeded

	// Log webhook received
	fmt.Printf("Webhook received from %s: payment_id=%s, event_type=%s\n",
		event.ProviderName, event.ProviderChargeID, event.EventType)

	// Return 200 OK to acknowledge
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "webhook received",
	})
}

// RegisterRoutes registers webhook routes
func (h *Handler) RegisterRoutes(router interface {
	Post(pattern string, handlerFn http.HandlerFunc)
}) {
	// Type assertion to chi.Router would go here in real code
	// For now, routes would be registered manually in main.go
}
