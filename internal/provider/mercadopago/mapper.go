package mercadopago

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/DelgadoElias/billax/internal/provider"
)

// mapStatus converts Mercado Pago payment status to payd's internal ChargeStatus
func mapStatus(mpStatus string) provider.ChargeStatus {
	switch mpStatus {
	case "approved":
		return provider.ChargeStatusSucceeded
	case "pending", "in_process", "authorized":
		return provider.ChargeStatusPending
	case "rejected", "cancelled", "charged_back", "refunded":
		return provider.ChargeStatusFailed
	default:
		// Safe fallback for unknown statuses — treat as pending to avoid false failures
		return provider.ChargeStatusPending
	}
}

// mapRefundStatus converts Mercado Pago refund status to ChargeStatus
func mapRefundStatus(mpStatus string) provider.ChargeStatus {
	switch mpStatus {
	case "approved":
		return provider.ChargeStatusSucceeded
	case "pending":
		return provider.ChargeStatusPending
	default:
		return provider.ChargeStatusFailed
	}
}

// mapPaymentMethod extracts payment method info from a Mercado Pago payment
// Only stores non-sensitive data: brand, type, last 4 digits
func mapPaymentMethod(p *mpPayment) *provider.PaymentMethodInfo {
	if p == nil {
		return nil
	}

	info := &provider.PaymentMethodInfo{
		Type:  p.PaymentTypeID, // "credit_card", "debit_card", "account_money", etc.
		Brand: p.PaymentMethodID, // "visa", "mastercard", "mercadopago", etc.
	}

	if p.Card != nil && p.Card.LastFourDigits != "" {
		info.LastFour = p.Card.LastFourDigits
	}

	return info
}

// buildCreatePaymentRequest converts payd's ChargeRequest to Mercado Pago's request format
func buildCreatePaymentRequest(req provider.ChargeRequest) mpCreatePaymentRequest {
	return mpCreatePaymentRequest{
		TransactionAmount: centavosToUnits(req.Amount), // Convert centavos to primary currency unit
		Description:       req.Description,
		ExternalReference: req.IdempotencyKey, // Used for idempotency and reconciliation
		Payer:             mpPayer{Email: req.ExternalCustomerID}, // MP requires payer as nested object
		Metadata:          req.Metadata,
	}
}

// mapToChargeResult converts a Mercado Pago payment response to payd's ChargeResult
func mapToChargeResult(p *mpPayment) (*provider.ChargeResult, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshaling raw payment response: %w", err)
	}

	return &provider.ChargeResult{
		ProviderChargeID: strconv.FormatInt(p.ID, 10), // MP IDs are int64
		Status:           mapStatus(p.Status),
		FailureReason:    p.StatusDetail, // Only meaningful when status is "rejected"
		PaymentMethod:    mapPaymentMethod(p),
		RawResponse:      raw,
	}, nil
}

// mapToRefundResult converts a Mercado Pago refund response to payd's RefundResult
func mapToRefundResult(r *mpRefund) (*provider.RefundResult, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshaling raw refund response: %w", err)
	}

	return &provider.RefundResult{
		ProviderRefundID: strconv.FormatInt(r.ID, 10),
		Status:           mapRefundStatus(r.Status),
		RawResponse:      raw,
	}, nil
}
