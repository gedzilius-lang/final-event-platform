package domain

import "errors"

var (
	ErrInvalidEventType     = errors.New("invalid or unrecognised event_type")
	ErrMissingUserID        = errors.New("user_id is required")
	ErrZeroAmount           = errors.New("amount_nc must not be zero")
	ErrMissingIdempotencyKey = errors.New("idempotency_key is required")
	ErrUnauthorisedWriter   = errors.New("caller is not authorised to write this event_type")
	ErrNotFound             = errors.New("not found")
)
