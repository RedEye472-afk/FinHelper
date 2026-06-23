// Package pii masks personally identifiable information that may leak through
// free-text fields (operation counterparty / description, bank statement
// memos) before it is persisted or logged.
//
// PRIVACY_RULES.md §"Маскирование PII":
//   - ФИО → [PERSON]
//   - Телефоны → [PHONE]
//   - Медицинские категории → [MEDICAL]
//   - Юридические категории → [LEGAL]
//
// The masking is conservative: it never rewrites digits that look like amounts
// or dates, and it is idempotent (masking already-masked text is a no-op).
package pii

import "regexp"

// Mask rewrites the input, replacing detected PII with bracketed placeholders.
// The input may be empty; the empty string is returned unchanged.
func Mask(s string) string {
	if s == "" {
		return s
	}
	for _, r := range maskers {
		s = r.ReplaceAllString(s, r.replacement)
	}
	return s
}

// masker pairs a compiled regex with its replacement token.
type masker struct {
	*regexp.Regexp
	replacement string
}

// NOTE on word boundaries: Go's regexp (RE2) treats \b as an ASCII-only
// boundary — it does NOT fire next to Cyrillic letters, because \w is
// [0-9A-Za-z_] in RE2. So for Cyrillic keyword rules we deliberately omit \b.
// Order matters too: digit/ASCII rules run first, then ALLCAPS person matching
// (idempotent because placeholders are wrapped in []), then keyword rules.
var maskers = []masker{
	// Russian phone numbers: +7 (XXX) XXX-XX-XX, 8XXXXXXXXXX, etc.
	{
		regexp.MustCompile(`(?i)(?:\+7|8|7)\s*\(?\d{3}\)?[\s\-]?\d{3}[\s\-]?\d{2}[\s\-]?\d{2}`),
		"[PHONE]",
	},
	// Email addresses — these are PII even when they appear inside a memo.
	{
		regexp.MustCompile(`(?i)[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}`),
		"[EMAIL]",
	},
	// Passport-like series/number: "12 34 567890" or "1234-567890".
	{
		regexp.MustCompile(`\d{2}\s?\d{2}\s?\d{6}`),
		"[PASSPORT]",
	},
	// Card-like numbers: 13–19 digit runs (optionally space/dash separated).
	// Anchored on a leading non-digit so we don't eat the kopecks of an amount.
	{
		regexp.MustCompile(`(?:^|[^\d.])(\d[ -]?){13,19}`),
		" [CARD]",
	},
	// Person names — Cyrillic ALLCAPS. A token is either a full name
	// ([А-ЯЁ]{2,}) or an initial ([А-ЯЁ]\.). We require 2+ such tokens
	// separated by whitespace, so a single ALLCAPS word ("АТМ", "Чек") is
	// NOT masked. Idempotent: placeholders are wrapped in [] which breaks the
	// whitespace-only separator.
	{
		regexp.MustCompile(`(?:[А-ЯЁ]{2,}|[А-ЯЁ]\.)(?:\s+(?:[А-ЯЁ]{2,}|[А-ЯЁ]\.))+`),
		"[PERSON]",
	},
	// Person names — Latin ALLCAPS, same token logic.
	{
		regexp.MustCompile(`(?:[A-Z]{2,}|[A-Z]\.)(?:\s+(?:[A-Z]{2,}|[A-Z]\.))+`),
		"[PERSON]",
	},
	// Medical merchants/categories (case-insensitive; Cyrillic, no \b).
	{
		regexp.MustCompile(`(?i)аптек[а-я]*|медицин[а-я]*|клиник[а-я]*|стоматолог[а-я]*|ветеринар[а-я]*|pharmac[a-z]*|clinic[a-z]*|medical[a-z]*|dent(?:ist|al)[a-z]*`),
		"[MEDICAL]",
	},
	// Legal/notary services (case-insensitive; Cyrillic, no \b).
	{
		regexp.MustCompile(`(?i)нотариус[а-я]*|адвокат[а-я]*|юрист[а-я]*|юридическ[а-я]*|notary|attorney|legal\s+services?`),
		"[LEGAL]",
	},
}
