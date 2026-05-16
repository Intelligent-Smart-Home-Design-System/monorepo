package catalogbuilder

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

type taxonomySchemas struct {
	compiled map[string]*gojsonschema.Schema
}

func loadTaxonomySchemas(path string) (*taxonomySchemas, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var raw map[string]struct {
		Schema json.RawMessage `json:"schema"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	compiled := make(map[string]*gojsonschema.Schema)
	for category, entry := range raw {
		if len(entry.Schema) == 0 {
			continue
		}
		s, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(entry.Schema))
		if err != nil {
			return nil, fmt.Errorf("compile schema for %q: %w", category, err)
		}
		compiled[category] = s
	}
	return &taxonomySchemas{compiled: compiled}, nil
}

// validate returns whether attrs conforms to the schema for category.
// Returns true (valid) if no schema is registered for the category.
func (t *taxonomySchemas) validate(category string, attrs map[string]any) (bool, []string) {
	schema, ok := t.compiled[category]
	if !ok {
		return true, nil
	}

	result, err := schema.Validate(gojsonschema.NewGoLoader(attrs))
	if err != nil {
		return false, []string{err.Error()}
	}
	if !result.Valid() {
		errs := make([]string, len(result.Errors()))
		for i, e := range result.Errors() {
			errs[i] = e.String()
		}
		return false, errs
	}
	return true, nil
}
