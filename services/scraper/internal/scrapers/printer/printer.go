package printer

import (
	"context"
	"net/http"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type printerScraper struct {
}

func (p *printerScraper) Scrape(ctx context.Context, task domain.ScrapeTask) (domain.ScrapeResult, error) {
	return domain.ScrapeResult{
		Resources: []domain.Resource{
			{
				Name:         "url",
				StatusCode:   http.StatusOK,
				ResponseBody: []byte(task.URL),
			},
			{
				Name:         "page_type",
				StatusCode:   http.StatusOK,
				ResponseBody: []byte(task.PageType),
			},
		},
	}, nil
}

func NewPrinterScraper() *printerScraper {
	return &printerScraper{}
}
