package apify

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
    "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

type ApifyItem struct {
    Title        string   `json:"title"`
    Price        int      `json:"price"`
    Rating       float64  `json:"rating"`
    ReviewCount  int      `json:"reviewCount"`
    SellerName   string   `json:"sellerName"`
    Availability string   `json:"availability"`
    Images       []string `json:"images"`
    Brand        string   `json:"brand"`
}

func ParseApifyResult(jsonData []byte, brandAliases map[string]string, snapshotID int) ([]*domain.ListingParseResult, error) {
    var items []ApifyItem
    if err := json.Unmarshal(jsonData, &items); err != nil {
        return nil, err
    }
    results := make([]*domain.ListingParseResult, 0, len(items))
    for _, item := range items {
        res := &domain.ListingParseResult{
            PageSnapshotID:      snapshotID,
            ParsedAt:            time.Now(),
            ExtractorVer:        "apify_v1",
            HasSmartHomeMarkers: true,
            Name:                item.Title,
            Brand:               parser.NormalizeBrand(item.Brand, brandAliases),
            Rating:              item.Rating,
            ReviewCount:         item.ReviewCount,
            InStock:             strings.Contains(strings.ToLower(item.Availability), "в наличии"),
        }
        if item.Price > 0 {
            res.Price = &item.Price
            currency := "RUB"
            res.Currency = &currency
        }
        if len(item.Images) > 0 {
            res.ImageURL = item.Images[0]
        }

        res.Text = strings.Join([]string{item.Title, item.Brand, item.SellerName}, " | ")

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