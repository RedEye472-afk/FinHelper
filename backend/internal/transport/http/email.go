package http

import (
	"errors"
	"net/mail"
)

const maxEmailLen = 254 // RFC 5321

// validateEmail rejects obvious garbage without false-positiving valid
// addresses. mail.ParseAddress is stricter than a regex and Unicode-aware,
// but it also accepts "Name <addr>"; we require a bare addr-spec.
func validateEmail(s string) error {
	if len(s) == 0 || len(s) > maxEmailLen {
		return errors.New("email must be 1..254 chars")
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return errors.New("invalid email format")
	}
	if addr.Address != s {
		return errors.New("email must be a bare address without display name")
	}
	return nil
}
