package catalogbuilder

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"
	"github.com/rs/zerolog"
)

var errNoModel = errors.New("listing has no model")
var errNoAttrs = errors.New("no identifying attributes to use")
var errAttrNotFound = errors.New("attribute not found")
var errUnsupportedAttr = errors.New("unsupported attribute type")

func getPrimaryKey(listing *domain.ExtractedListing) (string, error) {
	if listing.Model == nil || *listing.Model == "unknown" {
		return "", errNoModel
	}
	return fmt.Sprintf("%s:%s", listing.Category, *listing.Model), nil
}

func getSecondaryKey(listing *domain.ExtractedListing, attrs []string) (string, error) {
	if len(attrs) == 0 {
		return "", errNoAttrs
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s:%s", listing.Brand, listing.Category)
	for _, name := range attrs {
		attr, ok := listing.DeviceAttributes[name]
		if !ok {
			return "", fmt.Errorf("%w: %s", errAttrNotFound, name)
		}
		attrStr, err := attributeToString(attr)
		if err != nil {
			return "", err
		}
		sb.WriteString(":")
		sb.WriteString(attrStr)
	}
	return sb.String(), nil
}

func attributeToString(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%.1f", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case []any:
		strs := make([]string, 0, len(v))
		for _, s := range v {
			if s, ok := s.(string); ok {
				strs = append(strs, s)
			}
		}
		slices.Sort(strs)
		sb := strings.Builder{}
		for i, s := range strs {
			sb.WriteString(s)
			if i != len(strs)-1 {
				sb.WriteRune(',')
			}
		}
		return sb.String(), nil
	case nil:
		return "null", nil
	default:
		return "", fmt.Errorf("%w: %T", errUnsupportedAttr, val)
	}
}

func deduplicateAttributes(cluster []*domain.ExtractedListing, log zerolog.Logger) map[string]any {
	if len(cluster) == 0 {
		return nil
	}

	// collect all field names seen across all listings
	allFields := make(map[string]struct{})
	for _, listing := range cluster {
		for k := range listing.DeviceAttributes {
			allFields[k] = struct{}{}
		}
	}

	result := make(map[string]any)

	for field := range allFields {
		values := make([]any, 0, len(cluster))
		for _, listing := range cluster {
			if v, ok := listing.DeviceAttributes[field]; ok && v != nil {
				values = append(values, v)
			}
		}

		if len(values) == 0 {
			// all null - omit field
			continue
		}

		merged, err := mergeFieldValues(values)
		if err != nil {
			log.Warn().Str("field", field).Int("listing_id", cluster[0].Id).Err(err).Msg("could not merge field values, skipping field")
			continue
		}
		result[field] = merged
	}

	return result
}

func mergeFieldValues(values []any) (any, error) {
	if len(values) == 0 {
		return nil, nil
	}

	switch values[0].(type) {
	case bool:
		return medianBool(values), nil

	case float64: // JSON numbers unmarshal as float64
		return medianFloat(values), nil

	case string:
		return mostFrequentString(values), nil

	case []any:
		return mergeStringArrays(values), nil

	default:
		return nil, fmt.Errorf("%w: %T", errUnsupportedAttr, values[0])
	}
}

func medianBool(values []any) bool {
	trueCount := 0
	for _, v := range values {
		if b, ok := v.(bool); ok && b {
			trueCount++
		}
	}
	return trueCount*2 >= len(values)
}

func medianFloat(values []any) float64 {
	floats := make([]float64, 0, len(values))
	for _, v := range values {
		if f, ok := v.(float64); ok {
			floats = append(floats, f)
		}
	}
	sort.Float64s(floats)
	return floats[(len(floats)+1)/2-1]
}

func mostFrequentString(values []any) string {
	freq := make(map[string]int)
	for _, v := range values {
		if s, ok := v.(string); ok {
			freq[s]++
		}
	}
	best := ""
	bestCount := 0
	for s, count := range freq {
		if count > bestCount {
			best = s
			bestCount = count
		}
	}
	return best
}

// mergeStringArrays treats each []string as a set, includes value if majority of listings have it
// also ignores listings that don't have any values in that set
func mergeStringArrays(values []any) []string {
	valueCount := make(map[string]int)
	numNonEmptyValues := 0
	for _, v := range values {
		arr, ok := v.([]any)
		if !ok {
			continue
		}
		if len(arr) > 0 {
			numNonEmptyValues++
		}
		seen := make(map[string]bool)
		for _, item := range arr {
			if s, ok := item.(string); ok && !seen[s] {
				valueCount[s]++
				seen[s] = true
			}
		}
	}

	var result []string
	for s, count := range valueCount {
		if count*2 >= numNonEmptyValues {
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}
