package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSourceJobFilter_Matches_TrackedPageIDs(t *testing.T) {
	f := SourceJobFilter{TrackedPageIDs: []int{42, 99}}
	assert.True(t, f.Matches(42, "https://example.com/any"))
	assert.False(t, f.Matches(1, "https://example.com/any"))
}

func TestSourceJobFilter_Matches_URLContains(t *testing.T) {
	f := SourceJobFilter{URLContains: []string{"zigbee", "datchik"}}
	assert.True(t, f.Matches(1, "https://www.dns-shop.ru/product/abc/datcik-zigbee/"))
	assert.False(t, f.Matches(1, "https://www.dns-shop.ru/product/abc/televizor/"))
}

func TestSourceJobFilter_Matches_Both(t *testing.T) {
	f := SourceJobFilter{
		TrackedPageIDs: []int{10},
		URLContains:    []string{"datchik"},
	}
	assert.True(t, f.Matches(10, "https://example.com/datchik"))
	assert.False(t, f.Matches(10, "https://example.com/televizor"))
	assert.False(t, f.Matches(11, "https://example.com/datchik"))
}

func TestSourceJobFilter_MatchesSnapshot_PageSnapshotIDs(t *testing.T) {
	f := SourceJobFilter{PageSnapshotIDs: []int{100, 200}}
	now := time.Now()
	assert.True(t, f.MatchesSnapshot(100, 1, "https://example.com/a", now))
	assert.False(t, f.MatchesSnapshot(99, 1, "https://example.com/a", now))
}

func TestSourceJobFilter_MatchesSnapshot_BothIDTypes(t *testing.T) {
	f := SourceJobFilter{
		PageSnapshotIDs: []int{100},
		TrackedPageIDs:  []int{42},
	}
	now := time.Now()
	assert.True(t, f.MatchesSnapshot(100, 42, "https://example.com", now))
	assert.False(t, f.MatchesSnapshot(100, 99, "https://example.com", now))
}

func TestSourceJobFilter_MatchesScrapedTime_ParseFromOnly(t *testing.T) {
	after := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	f := SourceJobFilter{ScrapedAfter: after}

	assert.True(t, f.MatchesSnapshot(1, 1, "https://x", after))
	assert.True(t, f.MatchesSnapshot(1, 1, "https://x", after.Add(24*time.Hour)))
	assert.False(t, f.MatchesSnapshot(1, 1, "https://x", after.Add(-time.Hour)))
}

func TestSourceJobFilter_MatchesScrapedTime_ParseRange(t *testing.T) {
	after := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	f := SourceJobFilter{ScrapedAfter: after, ScrapedBefore: before}

	assert.True(t, f.MatchesSnapshot(1, 1, "https://x", after))
	assert.True(t, f.MatchesSnapshot(1, 1, "https://x", before))
	assert.False(t, f.MatchesSnapshot(1, 1, "https://x", before.Add(time.Hour)))
}

func TestSourceJobFilter_MatchesTask_Unscraped(t *testing.T) {
	after := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	f := SourceJobFilter{ScrapedAfter: after}
	seen := after.Add(-time.Hour)

	assert.False(t, f.MatchesTask(1, "https://x", seen, nil))

	fNoLower := SourceJobFilter{ScrapedBefore: after.Add(24 * time.Hour)}
	assert.True(t, fNoLower.MatchesTask(1, "https://x", seen, nil))
}

func TestSourceJobFilter_MatchesTask_LastScraped(t *testing.T) {
	after := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	f := SourceJobFilter{ScrapedAfter: after}
	ts := after.Add(2 * time.Hour)
	seen := after.Add(-24 * time.Hour)

	assert.True(t, f.MatchesTask(1, "https://x", seen, &ts))
	assert.False(t, f.MatchesTask(1, "https://x", seen, ptrTime(after.Add(-time.Hour))))
}

func TestSourceJobFilter_MatchesTask_CreatedToday(t *testing.T) {
	start := time.Date(2026, 6, 29, 0, 0, 0, 0, time.FixedZone("MSK", 3*3600))
	f := SourceJobFilter{CreatedAfter: start}
	oldPage := start.Add(-48 * time.Hour)
	newPage := start.Add(2 * time.Hour)

	assert.True(t, f.MatchesTask(1, "https://x", newPage, nil))
	assert.False(t, f.MatchesTask(2, "https://y", oldPage, nil))
}

func TestSourceJobFilter_BootstrapMode(t *testing.T) {
	seed, db := SourceJobFilter{}.BootstrapMode()
	assert.True(t, seed)
	assert.True(t, db)

	seed, db = SourceJobFilter{DiscoveryBootstrap: []string{"seed"}}.BootstrapMode()
	assert.True(t, seed)
	assert.False(t, db)

	seed, db = SourceJobFilter{DiscoveryBootstrap: []string{"db"}}.BootstrapMode()
	assert.False(t, seed)
	assert.True(t, db)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
