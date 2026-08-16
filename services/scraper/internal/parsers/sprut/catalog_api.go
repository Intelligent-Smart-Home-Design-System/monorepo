package sprut

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
)

// api.sprut.ai backs the Nuxt frontend; catalog listing pages (product-grid, e.g.
// /catalog/light/lightbulb) 500 on direct SSR load, so listing discovery goes through
// this JSON API instead of scraping that HTML. Hub pages (/catalog/section, /catalog/light)
// load fine and are still parsed as HTML — see browse.go.
const apiOrigin = "https://api.sprut.ai"

type catalogSlugResponse struct {
	Item struct {
		ID int `json:"id"`
	} `json:"item"`
}

type catalogItemsResponse struct {
	Items []struct {
		Slug string `json:"slug"`
	} `json:"items"`
	Meta struct {
		TotalCount int `json:"totalCount"`
		PageCount  int `json:"pageCount"`
	} `json:"_meta"`
}

func catalogItemsURL(catalogID, page int) string {
	v := url.Values{}
	v.Set("filter[catalog_id]", strconv.Itoa(catalogID))
	v.Set("paginate[page]", strconv.Itoa(page))
	v.Set("sort[rating]", "desc")
	return apiOrigin + "/catalogs/items?" + v.Encode()
}

func fetchJSON(ctx context.Context, scraperInst scraper.Scraper, apiURL string, out interface{}) error {
	result, err := scraperInst.Scrape(ctx, domain.ScrapeTask{URL: apiURL})
	if err != nil {
		return err
	}
	if len(result.Resources) == 0 {
		return fmt.Errorf("empty response for %s", apiURL)
	}
	if err := json.Unmarshal(result.Resources[0].ResponseBody, out); err != nil {
		return fmt.Errorf("decode %s: %w", apiURL, err)
	}
	return nil
}

// resolveCatalogID looks up the numeric catalog_id for a URL's slug (e.g. "lightbulb"),
// used both to check whether a node has items and to build the items API URL.
func resolveCatalogID(ctx context.Context, scraperInst scraper.Scraper, slug string) (int, error) {
	var resp catalogSlugResponse
	apiURL := apiOrigin + "/catalogs/slug/" + url.PathEscape(slug) + "?expand=seo"
	if err := fetchJSON(ctx, scraperInst, apiURL, &resp); err != nil {
		return 0, err
	}
	if resp.Item.ID == 0 {
		return 0, fmt.Errorf("catalog not found for slug %q", slug)
	}
	return resp.Item.ID, nil
}

func fetchItemsPage(ctx context.Context, scraperInst scraper.Scraper, catalogID, page int) (*catalogItemsResponse, error) {
	var resp catalogItemsResponse
	if err := fetchJSON(ctx, scraperInst, catalogItemsURL(catalogID, page), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
