package wildberries

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildDiscoverySearchURL(t *testing.T) {
	template := "https://www.wildberries.ru/__internal/u-search/exactmatch/ru/common/v18/search?page={page}&query={query}&resultset=catalog"
	url := BuildDiscoverySearchURL(template, "умная лампа", 2)
	assert.Equal(
		t,
		"https://www.wildberries.ru/__internal/u-search/exactmatch/ru/common/v18/search?page=2&query=%D1%83%D0%BC%D0%BD%D0%B0%D1%8F+%D0%BB%D0%B0%D0%BC%D0%BF%D0%B0&resultset=catalog",
		url,
	)
}
