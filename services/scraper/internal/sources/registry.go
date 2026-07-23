package sources

import (
	"fmt"
	"slices"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/printer"
)

// Registry holds built-in sources keyed by name.
type Registry map[string]Source

var builtinOrder = []string{
	domain.SourceDns,
	domain.SourceWildberries,
	domain.SourceYandex,
	domain.SourceSprut,
	domain.SourcePrinter,
	domain.SourceApifyYandexMarket,
}

// NewScraper builds a scraper for one source (factory entry point).
func NewScraper(source string, cfg config.Config, log zerolog.Logger) (scraper.Scraper, error) {
	src, err := newSource(source, cfg, log)
	if err != nil {
		return nil, err
	}
	return src.Scraper(), nil
}

// NewRegistry constructs all built-in sources.
func NewRegistry(cfg config.Config, log zerolog.Logger) (Registry, error) {
	out := make(Registry, len(builtinOrder))
	for _, name := range builtinOrder {
		src, err := newSource(name, cfg, log)
		if err != nil {
			return nil, err
		}
		out[name] = src
	}
	return out, nil
}

func newSource(name string, cfg config.Config, log zerolog.Logger) (Source, error) {
	switch name {
	case domain.SourcePrinter:
		return Printer{Base: Base{name: domain.SourcePrinter, scraper: printer.NewPrinterScraper()}}, nil
	case domain.SourceWildberries:
		return newWildberries(cfg, log), nil
	case domain.SourceYandex:
		return newYandex(cfg, log), nil
	case domain.SourceDns:
		return newDNS(cfg, log), nil
	case domain.SourceApifyYandexMarket:
		return newApify(cfg, log), nil
	case domain.SourceSprut:
		return newSprut(cfg, log), nil
	default:
		return nil, fmt.Errorf("unknown source %q", name)
	}
}

func (r Registry) ScraperMap() map[string]scraper.Scraper {
	out := make(map[string]scraper.Scraper, len(r))
	for name, src := range r {
		out[name] = src.Scraper()
	}
	return out
}

func (r Registry) Selected(names []string) []Source {
	if len(names) == 0 {
		return r.inOrder()
	}
	var out []Source
	for _, name := range builtinOrder {
		if slices.Contains(names, name) && r[name] != nil {
			out = append(out, r[name])
		}
	}
	return out
}

func (r Registry) inOrder() []Source {
	out := make([]Source, 0, len(builtinOrder))
	for _, name := range builtinOrder {
		if src, ok := r[name]; ok {
			out = append(out, src)
		}
	}
	return out
}

// Printer is a no-op discovery source (debug).
type Printer struct{ Base }
