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

type traitDef struct {
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
}

type typeDef struct {
	Traits               []string          `json:"traits"`
	PropertyDescriptions map[string]string `json:"property_descriptions"`
	ExtraSchema          struct {
		Properties map[string]interface{} `json:"properties"`
		Required   []string               `json:"required"`
	} `json:"extra_schema"`
}

type rawTaxonomy struct {
	Traits map[string]traitDef `json:"traits"`
	Types  map[string]typeDef  `json:"types"`
}

func loadTaxonomySchemas(path string) (*taxonomySchemas, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var raw rawTaxonomy
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	compiled := make(map[string]*gojsonschema.Schema)

	for category, tdef := range raw.Types {
		mergedProperties := make(map[string]interface{})
		var mergedRequired []string

		for _, traitName := range tdef.Traits {
			if trait, ok := raw.Traits[traitName]; ok {
				for propName, propVal := range trait.Properties {
					mergedProperties[propName] = propVal
				}
				mergedRequired = append(mergedRequired, trait.Required...)
			}
		}

		for propName, propVal := range tdef.ExtraSchema.Properties {
			mergedProperties[propName] = propVal
		}
		mergedRequired = append(mergedRequired, tdef.ExtraSchema.Required...)

		schemaMap := map[string]interface{}{
			"$schema":    "http://json-schema.org/draft-07/schema#",
			"type":       "object",
			"properties": mergedProperties,
			"required":   mergedRequired,
		}

		loader := gojsonschema.NewGoLoader(schemaMap)
		s, err := gojsonschema.NewSchema(loader)
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
