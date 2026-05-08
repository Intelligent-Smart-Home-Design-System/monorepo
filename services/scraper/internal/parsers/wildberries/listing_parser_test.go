package wildberries

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListingParser_Parse(t *testing.T) {
	detailJSON := `{
		"products": [{
			"id": 185192863,
			"brand": "Яндекс",
			"name": "Датчик протечки, Zigbee",
			"reviewRating": 4.7,
			"feedbacks": 212,
			"sizes": [{
				"price": {"product": 192800},
				"stocks": [{"qty": 37}]
			}]
		}]
	}`

	cardJSON := `{
		"nm_id": 185192863,
		"imt_name": "Датчик протечки Zigbee",
		"description": "Беспроводной датчик протечки",
		"subj_name": "Датчик",
		"selling": {"brand_name": "Yandex"},
		"options": [
			{"name": "Модель", "value": "YNDX-00558"},
			{"name": "Комплектация", "value": "2 шт."}
		],
		"contents": "Комплект: 2 шт."
	}`

	files := []*parser.ArchiveFile{
		{Name: "detail.json", Data: []byte(detailJSON)},
		{Name: "card.json", Data: []byte(cardJSON)},
	}

	p := NewListingParser()
	result, err := p.Parse(123, files)
	require.NoError(t, err)

	assert.Equal(t, 123, result.PageSnapshotID)
	assert.Equal(t, "Датчик протечки, Zigbee", result.Name)
	assert.Equal(t, "yandex", result.Brand)
	assert.Equal(t, 1928, *result.Price)
	assert.Equal(t, "RUB", *result.Currency)
	assert.Equal(t, "YNDX-00558", *result.ModelNumber)
	assert.Equal(t, "Датчик", *result.Category)
	assert.Equal(t, 2, *result.Quantity)
	assert.Equal(t, "2 шт.", *result.QuantityRaw)
	assert.Equal(t, 4.7, result.Rating)
	assert.Equal(t, 212, result.ReviewCount)
	assert.True(t, result.InStock)
	assert.Contains(t, result.Text, "Беспроводной датчик протечки")
	assert.NotEmpty(t, result.ContentHash)
}

func TestNormalizeBrand(t *testing.T) {
	assert.Equal(t, "yandex", normalizeBrand("  Yandex  "))
	assert.Equal(t, "apple-iphone", normalizeBrand("Apple iPhone"))
}

func TestExtractQuantity(t *testing.T) {
	opts := []cardOption{{Name: "Комплектация", Value: "3 шт."}}
	qty, raw := extractQuantity("", opts)
	assert.Equal(t, 3, qty)
	assert.Equal(t, "3 шт.", raw)

	opts2 := []cardOption{{Name: "Комплектация", Value: "Set of 2"}}
	qty2, raw2 := extractQuantity("", opts2)
	assert.Equal(t, 2, qty2)
	assert.Equal(t, "Set of 2", raw2)

	qty3, raw3 := extractQuantity("без комплектации", nil)
	assert.Equal(t, 1, qty3)
	assert.Equal(t, "без комплектации", raw3)
}