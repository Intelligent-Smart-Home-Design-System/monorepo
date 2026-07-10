package dns

import "testing"

func TestAnalyzeProbeResponse_SSRWithLinks(t *testing.T) {
	body := []byte(`<html><head><title>Search</title></head><body>
	<a href="/product/abc123/item-one/">item</a>
	<a href="/product/def456/item-two/">item2</a>
	</body></html>`)

	result := AnalyzeProbeResponse("https://example/search", 200, "text/html", body)
	if result.Case != FetchCaseSSRWithLinks {
		t.Fatalf("case = %q, want %q", result.Case, FetchCaseSSRWithLinks)
	}
	if len(result.ProductLinks) != 2 {
		t.Fatalf("product links = %d, want 2", len(result.ProductLinks))
	}
}

func TestAnalyzeProbeResponse_EmptyShell(t *testing.T) {
	body := []byte(`<html><head><title>DNS</title></head><body><div id="root"></div><script>window.__NUXT__={}</script></body></html>`)
	result := AnalyzeProbeResponse("https://example/", 200, "text/html", body)
	if result.Case != FetchCaseEmptyShell {
		t.Fatalf("case = %q, want %q (%s)", result.Case, FetchCaseEmptyShell, result.CaseReason)
	}
}

func TestAnalyzeProbeResponse_Blocked(t *testing.T) {
	body := []byte(`<html><body><div class="g-recaptcha"></div></body></html>`)
	result := AnalyzeProbeResponse("https://example/", 403, "text/html", body)
	if result.Case != FetchCaseBlocked {
		t.Fatalf("case = %q, want %q", result.Case, FetchCaseBlocked)
	}
}
