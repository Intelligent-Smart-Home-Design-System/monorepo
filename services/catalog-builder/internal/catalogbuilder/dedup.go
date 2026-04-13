package catalogbuilder

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"
)

var errNoModel = errors.New("listing has no model")
var errNoAttrs = errors.New("no identifying attributes to use")
var errAttrNotFound = errors.New("attribute not found")
var errUnsupportedAttr = errors.New("unsupported attribute type")

func getPrimaryKey(listing *domain.ExtractedListing) (string, error) {
	if listing.Model == nil {
		return "", errNoModel
	}
	return fmt.Sprintf("%s:%s:%s", listing.Brand, listing.Category, *listing.Model), nil
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
	default:
		return "", fmt.Errorf("%w: %T", errUnsupportedAttr, val)
	}
}
