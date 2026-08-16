package cli

import (
	"slices"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/rs/zerolog"
)

func withSource(logger zerolog.Logger, source string) zerolog.Logger {
	if source == "" {
		return logger
	}
	return logger.With().Str("source", source).Logger()
}

func logJobStart(logger zerolog.Logger, command string, sources, pageTypes []string, extra ...func(*zerolog.Event)) {
	evt := logger.Info().Str("command", command)
	if len(sources) > 0 {
		evt = evt.Strs("sources", sources)
	} else {
		evt = evt.Str("sources", "all")
	}
	if len(pageTypes) > 0 {
		evt = evt.Strs("page_types", pageTypes)
	}
	for _, apply := range extra {
		apply(evt)
	}
	evt.Msg("job started")
}

func uniqueTaskSources(tasks []domain.ScrapeTask) []string {
	seen := make(map[string]struct{}, len(tasks))
	var out []string
	for _, t := range tasks {
		if t.Source == "" {
			continue
		}
		if _, ok := seen[t.Source]; ok {
			continue
		}
		seen[t.Source] = struct{}{}
		out = append(out, t.Source)
	}
	slices.Sort(out)
	return out
}
