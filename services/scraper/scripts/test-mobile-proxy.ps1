# Quick mobile-proxy smoke test (DNS via user-mode Chrome + optional e2e).
#
# Usage (24h mobile proxy from Megafon panel — paste REAL login/password):
#   $env:SCRAPER_SCRAPING_PROXY = "http://iparchitect_XXXXX:YOUR_PASSWORD@188.143.169.27:30151"
#   cd services/scraper
#   .\scripts\test-mobile-proxy.ps1
#   .\scripts\test-mobile-proxy.ps1 -E2E

param(
    [switch]$E2E,
    [string]$Config = "cmd/scraper/config.mobile-proxy.toml"
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)

if (-not $env:SCRAPER_SCRAPING_PROXY) {
    Write-Host "Set proxy first (real login/password from provider, NOT the USER:PASS placeholder):" -ForegroundColor Yellow
    Write-Host '  $env:SCRAPER_SCRAPING_PROXY = "http://LOGIN:PASSWORD@host:PORT"' -ForegroundColor Cyan
    exit 1
}

# User-mode Chrome bypasses Qrator; config.mobile-proxy.toml also sets browser_user_mode=true.
$env:DNS_BROWSER_USER_MODE = "1"
if (-not $env:SCRAPER_SCRAPING_USER_AGENT) {
    $env:SCRAPER_SCRAPING_USER_AGENT = "Mozilla/5.0 (Linux; Android 14; Mobile) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"
}

$proxy = $env:SCRAPER_SCRAPING_PROXY
$redacted = $proxy -replace '(:)([^:@]+)(@)', ':****@'
Write-Host "Proxy: $redacted" -ForegroundColor Green
Write-Host "DNS browser: user-mode Chrome (DNS_BROWSER_USER_MODE=1)" -ForegroundColor Green

Write-Host ""
Write-Host "[0/4] Proxy HTTP check (ipify)..." -ForegroundColor Cyan
go run ./cmd/proxycheck -config $Config
if ($LASTEXITCODE -ne 0) {
    Write-Host "proxycheck failed - fix SCRAPER_SCRAPING_PROXY (real login/password, not USER:PASS from docs)" -ForegroundColor Red
    exit $LASTEXITCODE
}

Write-Host ""
Write-Host "[1/4] DNS catalog fetch via dnsdiag..." -ForegroundColor Cyan
go run ./cmd/dnsdiag -config $Config
if ($LASTEXITCODE -ne 0) {
    Write-Host "dnsdiag failed - check proxy, user-mode Chrome, and dns-shop availability" -ForegroundColor Red
    exit $LASTEXITCODE
}

Write-Host ""
Write-Host "[2/4] DNS e2e (warmup + listing scrape + parse)..." -ForegroundColor Cyan
if ($E2E) {
    go test -v -count=1 -timeout 8m -run TestDNSE2E_ListingScrapeParse ./internal/sources/
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} else {
    Write-Host "  skipped (pass -E2E to run, ~2-4 min)" -ForegroundColor DarkGray
}

Write-Host ""
Write-Host '[3/4] WB session mine (optional - opens Chrome)...' -ForegroundColor Cyan
Write-Host "  run manually if needed:" -ForegroundColor DarkGray
Write-Host "  go run ./cmd/wbsession -config cmd/scraper/config.wb-smoke.toml" -ForegroundColor DarkGray

Write-Host ""
Write-Host 'OK - mobile proxy smoke passed. Continue with config.mobile-proxy.toml' -ForegroundColor Green
