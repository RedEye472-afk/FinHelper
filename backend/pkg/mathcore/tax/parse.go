package tax

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// parseRules deserialises a tax_rules YAML document into Rules. The Dec
// fields implement yaml.Unmarshaler so the loader never touches float64.
func parseRules(raw []byte) (Rules, error) {
	var r Rules
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return Rules{}, fmt.Errorf("yaml: %w", err)
	}
	if r.Year == 0 {
		return Rules{}, fmt.Errorf("tax: rules missing required `year` field")
	}
	if r.Method == "" {
		return Rules{}, fmt.Errorf("tax: rules missing required `method` field")
	}
	return r, nil
}
