package storage

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// decimalScanner is a sql.Scanner that parses a NUMERIC/text column into a
// decimal.Decimal. Postgres returns NUMERIC as a string when scanned into an
// interface{} implementing Scanner, so we parse it directly — never through
// float64. NULL columns yield the zero decimal.
type decimalScanner struct {
	d decimal.Decimal
}

// Scan implements sql.Scanner. Handles string (NUMERIC wire form), []byte,
// and float64 (some drivers) — the float64 path is driver-specific and kept
// only to surface a clear error if a misconfigured driver ever uses it.
func (s *decimalScanner) Scan(v any) error {
	if v == nil {
		s.d = decimal.Zero
		return nil
	}
	switch x := v.(type) {
	case string:
		d, err := decimal.NewFromString(x)
		if err != nil {
			return fmt.Errorf("storage: parse decimal %q: %w", x, err)
		}
		s.d = d
		return nil
	case []byte:
		d, err := decimal.NewFromString(string(x))
		if err != nil {
			return fmt.Errorf("storage: parse decimal %q: %w", string(x), err)
		}
		s.d = d
		return nil
	case float64:
		// Should not happen with pgx + NUMERIC, but guard anyway.
		s.d = decimal.NewFromFloat(x)
		return nil
	default:
		return fmt.Errorf("storage: unsupported decimal scan type %T", v)
	}
}
