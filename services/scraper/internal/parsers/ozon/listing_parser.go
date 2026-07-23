package ozon

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
    "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

type Brand struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

type OzonItem struct {
    Title        string   `json:"title"`
    Price        string   `json:"price"`
    Rating       float64  `json:"rating"`
    ReviewCount  int      `json:"reviewCount"`
    SellerName   string   `json:"sellerName"`
    Availability string   `json:"availability"`
    Images       []string `json:"images"`
    Brand        interface{} `json:"brand"`
}

func ParseOzonResult(jsonData []byte, brandAliases map[string]string, snapshotID int) ([]*domain.ListingParseResult, error) {
    var items []OzonItem
    if err := json.Unmarshal(jsonData, &items); err != nil {
        return nil, err
    }
    results := make([]*domain.ListingParseResult, 0, len(items))
    for _, item := range items {
        brandName := ""
        switch v := item.Brand.(type) {
        case string:
            brandName = v
        case map[string]interface{}:
            if name, ok := v["name"].(string); ok {
                brandName = name
            }
        case nil:
            brandName = ""
        }
        if brandName == "" && item.SellerName != "" {
            brandName = item.SellerName
        }

        priceInt := 0
        if item.Price != "" {
            re := regexp.MustCompile(`[^0-9]`)
            cleaned := re.ReplaceAllString(item.Price, "")
            if cleaned != "" {
                if p, err := strconv.Atoi(cleaned); err == nil && p > 0 {
                    priceInt = p
                }
            }
        }

        res := &domain.ListingParseResult{
            PageSnapshotID:      snapshotID,
            ParsedAt:            time.Now(),
            ExtractorVer:        "ozon_v1",
            HasSmartHomeMarkers: true,
            Name:                item.Title,
            Brand:               parser.NormalizeBrand(brandName, brandAliases),
            Rating:              item.Rating,
            ReviewCount:         item.ReviewCount,
            InStock:             strings.Contains(strings.ToLower(item.Availability), "в наличии") || strings.Contains(strings.ToLower(item.Availability), "in stock"),
        }
        if priceInt > 0 {
            res.Price = &priceInt
            currency := "RUB"
            res.Currency = &currency
        }
        if len(item.Images) > 0 {
            res.ImageURL = item.Images[0]
        }
        res.Text = strings.Join([]string{item.Title, brandName, item.SellerName}, " | ")
        res.ContentHash = computeHash(res)
        results = append(results, res)
    }
    return results, nil
}

func computeHash(result *domain.ListingParseResult) string {
    data := fmt.Sprintf("%s|%s|%d|%v|%d|%d|%f",
        result.Name,
        result.Brand,
        nullIntOrDefault(result.Price),
        result.InStock,
        nullIntOrDefault(result.Quantity),
        result.ReviewCount,
        result.Rating,
    )
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

func nullIntOrDefault(ptr *int) int {
    if ptr == nil {
        return 0
    }
    return *ptr
}
