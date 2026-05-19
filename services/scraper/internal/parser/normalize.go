package parser

import "strings"

func NormalizeBrand(brand string, brandAliases map[string]string) string {
    if brand == "" {
        return ""
    }
    normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(brand)), " ", "-")
    if alias, ok := brandAliases[normalized]; ok {
        return alias
    }
    return normalized
}
