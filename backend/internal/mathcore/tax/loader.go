package tax

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
)

//go:embed configs/*.yaml
var builtinConfigs embed.FS

// LoadRules reads configs/tax_rules_{year}.yaml from the built-in embedded
// configs directory. Returns ErrUnsupportedYear if no config exists for the
// given year.
//
// Embedding (not runtime file IO) keeps the binary self-contained and makes
// the rules tamper-evident — changing rates requires a recompile, which is
// the right trade-off for a financial app where rules must be auditable.
//
// NOTE: embed.FS always uses forward slashes regardless of OS, so we join
// with path (not path/filepath) — the latter would emit "configs\…" on
// Windows and fail fs.ReadFile.
func LoadRules(year int) (Rules, error) {
	return LoadRulesFromFS(builtinConfigs, year)
}

// LoadRulesFromFS loads a year's rules from an arbitrary fs.FS whose root
// contains a configs/ directory. Exposed for testing with custom configs.
func LoadRulesFromFS(fsys fs.FS, year int) (Rules, error) {
	p := path.Join("configs", fmt.Sprintf("tax_rules_%d.yaml", year))
	raw, err := fs.ReadFile(fsys, p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Rules{}, fmt.Errorf("%w: %d", ErrUnsupportedYear, year)
		}
		return Rules{}, fmt.Errorf("tax: read %s: %w", p, err)
	}
	rules, err := parseRules(raw)
	if err != nil {
		return Rules{}, fmt.Errorf("tax: parse %s: %w", p, err)
	}
	if rules.Year != year {
		return Rules{}, fmt.Errorf("tax: %s declares year %s, expected %d",
			p, strconv.Itoa(rules.Year), year)
	}
	return rules, nil
}

// MustLoadRules is a convenience wrapper that panics on error. Use only for
// constants known at compile time (e.g. test fixtures).
func MustLoadRules(year int) Rules {
	r, err := LoadRules(year)
	if err != nil {
		panic(fmt.Sprintf("tax: load %d: %v", year, err))
	}
	return r
}
