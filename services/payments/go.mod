module niteos.internal/payments

go 1.22

require (
	github.com/lib/pq v1.10.9
	github.com/stripe/stripe-go/v76 v76.25.0
	niteos.internal/pkg v0.0.0
)

replace niteos.internal/pkg => ../../pkg
