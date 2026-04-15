package wildberries

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestExtractNmID(t *testing.T) {
	tests := []struct{
		url string
		expected int
		hasErr bool
	}{
		{"https://www.wildberries.ru/catalog/185192863/detail.aspx", 185192863, false},
		{"https://www.wildberries.ru/catalog/123/", 123, false},
		{"https://wrong.com", 0, true},
	}
	for _, tt := range tests {
		id, err := extractNmID(tt.url)
		if tt.hasErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, id)
		}
	}
}

func TestBuildCardURL(t *testing.T) {
	url := buildCardURL(12, 185192863)
	assert.Equal(t, "https://basket-12.wbbasket.ru/vol1851/part185192/185192863/info/ru/card.json", url)
}