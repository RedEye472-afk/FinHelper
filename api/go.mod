module github.com/RedEye472-afk/FinHelper/vercelapi

// Build cache invalidation: 2026-07-08 fresh deploy
go 1.26.4

require (
	github.com/RedEye472-afk/FinHelper v0.0.0-00010101000000-000000000000
	github.com/go-chi/chi/v5 v5.3.0
	github.com/go-chi/cors v1.2.2
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/text v0.38.0 // indirect
)

replace github.com/RedEye472-afk/FinHelper => ../backend
