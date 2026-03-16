// Package idempotency provides idempotency key generation for NiteOS services.
//
// Convention (from DATA_OWNERSHIP.md and LEDGER_AND_SETTLEMENT_SPEC.md):
//
//	{source}:{reference_id}:{event_type}
//
// Examples:
//
//	payments:pi_abc123:topup_confirmed
//	orders:ord_uuid:order_paid
//	edge:dev_uuid:ses_uuid:1001   (edge-sourced, includes sequence number)
package idempotency

import "fmt"

// Key returns a standard cloud-sourced idempotency key.
func Key(source, referenceID, eventType string) string {
	return fmt.Sprintf("%s:%s:%s", source, referenceID, eventType)
}

// EdgeKey returns an idempotency key for an edge-sourced event.
// Format: edge:{device_id}:{session_id}:{edge_seq}
func EdgeKey(deviceID, sessionID string, edgeSeq int64) string {
	return fmt.Sprintf("edge:%s:%s:%d", deviceID, sessionID, edgeSeq)
}

// WebhookKey returns an idempotency key for a payment provider webhook event.
// Format: {provider}:webhook:{provider_event_id}
func WebhookKey(provider, providerEventID string) string {
	return fmt.Sprintf("%s:webhook:%s", provider, providerEventID)
}
