module niteos.internal/catalog

go 1.22

require (
	github.com/lib/pq v1.10.9
	golang.org/x/crypto v0.21.0
	niteos.internal/pkg v0.0.0
)

replace niteos.internal/pkg => ../../pkg
