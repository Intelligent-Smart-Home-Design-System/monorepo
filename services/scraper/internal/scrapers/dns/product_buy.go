package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

const (
	productBuyURL      = "https://www.dns-shop.ru/ajax-state/product-buy/"
	productBuyFileName = "product-buy.json"
)

var (
	csrfTokenRE     = regexp.MustCompile(`name="csrf-token"\s+content="([^"]+)"`)
	productCodeRE   = regexp.MustCompile(`product-card-top__code-prefix">Код товара:\s*</span>\s*(\d+)`)
	productBuyHashRE = regexp.MustCompile(`"type":"product-buy","hash":"([a-f0-9]+)"`)
	productBuyIDRE  = regexp.MustCompile(`id="(as-[^"]+)"\s+class="product-buy"`)
)

type productBuyPayload struct {
	Type       string                  `json:"type"`
	Hash       string                  `json:"hash"`
	Containers []productBuyContainer   `json:"containers"`
}

type productBuyContainer struct {
	ID   string              `json:"id"`
	Data productBuyContainerData `json:"data"`
}

type productBuyContainerData struct {
	ID     string                 `json:"id"`
	Params productBuyContainerParams `json:"params"`
}

type productBuyContainerParams struct {
	ShowOneClick bool `json:"showOneClick"`
	IsCard       bool `json:"isCard"`
}

func extractProductBuyRequest(html []byte) (productBuyPayload, string, error) {
	csrfMatch := csrfTokenRE.FindSubmatch(html)
	if len(csrfMatch) < 2 {
		return productBuyPayload{}, "", fmt.Errorf("csrf token not found")
	}
	csrfToken := string(csrfMatch[1])

	codeMatch := productCodeRE.FindSubmatch(html)
	if len(codeMatch) < 2 {
		return productBuyPayload{}, "", fmt.Errorf("product code not found")
	}
	productCode := string(codeMatch[1])

	hashMatch := productBuyHashRE.FindSubmatch(html)
	if len(hashMatch) < 2 {
		return productBuyPayload{}, "", fmt.Errorf("product-buy hash not found")
	}
	hash := string(hashMatch[1])

	idMatch := productBuyIDRE.FindSubmatch(html)
	if len(idMatch) < 2 {
		return productBuyPayload{}, "", fmt.Errorf("product-buy container id not found")
	}
	containerID := string(idMatch[1])

	payload := productBuyPayload{
		Type: "product-buy",
		Hash: hash,
		Containers: []productBuyContainer{{
			ID: containerID,
			Data: productBuyContainerData{
				ID: productCode,
				Params: productBuyContainerParams{
					ShowOneClick: true,
					IsCard:       true,
				},
			},
		}},
	}
	return payload, csrfToken, nil
}

func buildProductBuyForm(payload productBuyPayload) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return url.Values{"data": {string(raw)}}.Encode(), nil
}

func (s *Scraper) fetchProductBuy(ctx context.Context, client *http.Client, pageURL string, html []byte) (*domain.Resource, error) {
	payload, csrfToken, err := extractProductBuyRequest(html)
	if err != nil {
		return nil, err
	}

	form, err := buildProductBuyForm(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, productBuyURL, strings.NewReader(form))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "https://www.dns-shop.ru")
	req.Header.Set("Referer", pageURL)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product-buy HTTP %d", resp.StatusCode)
	}

	return &domain.Resource{
		Name:         productBuyFileName,
		URL:          productBuyURL,
		ResponseBody: body,
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Timestamp:    time.Now(),
	}, nil
}
