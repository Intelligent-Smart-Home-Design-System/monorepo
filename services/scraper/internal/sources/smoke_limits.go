package sources

const (
	// SmokeMaxListings is how many listing URLs we validate in WB discovery smoke (no per-listing scrape).
	SmokeMaxListings = 10
	// SmokeMaxDNSFetches caps DNS BFS page fetches in smoke tests (warmup is separate).
	SmokeMaxDNSFetches = 10
)
