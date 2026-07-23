# WB smoke micro-steps. Run from services/scraper.
#
#   $env:SCRAPER_SCRAPING_PROXY = "http://LOGIN:PASSWORD@188.143.169.27:30151"
#   .\scripts\test-wb-smoke.ps1 -Step 2
#   .\scripts\test-wb-smoke.ps1 -E2E

param(
    [ValidateRange(1, 5)]
    [int]$Step = 0,
    [switch]$E2E,
    [string]$Config = "cmd/scraper/config.wb-smoke.toml"
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)

if (-not $env:SCRAPER_SCRAPING_PROXY) {
    Write-Host "Set proxy first (real login/password from Megafon panel, NOT USER:PASS from docs):" -ForegroundColor Yellow
    Write-Host '  $env:SCRAPER_SCRAPING_PROXY = "http://LOGIN:PASSWORD@host:PORT"' -ForegroundColor Cyan
    exit 1
}

$redacted = $env:SCRAPER_SCRAPING_PROXY -replace '(:)([^:@]+)(@)', ':****@'
Write-Host "Config: $Config" -ForegroundColor Green
Write-Host "Proxy:  $redacted" -ForegroundColor Green

Write-Host ""
Write-Host "Proxy HTTP check..." -ForegroundColor Cyan
go run ./cmd/proxycheck -config $Config
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

function Run-Step([int]$n, [string]$name) {
    Write-Host ""
    Write-Host "=== Step $n`: $name ===" -ForegroundColor Cyan
    go test -v -count=1 -timeout 12m -run "TestWBStep0$n" ./internal/sources/
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

if ($E2E) {
    Write-Host ""
    Write-Host "Mining WB session (visible Chrome)..." -ForegroundColor Cyan
    go run ./cmd/wbsession -config $Config -timeout 10m
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host ""
    Write-Host "Running full WB e2e (TestWildberriesSmoke)..." -ForegroundColor Cyan
    Write-Host "Artifacts -> testdata/wb-smoke-dump/" -ForegroundColor DarkGray
    go test -v -count=1 -timeout 12m -run TestWildberriesSmoke ./internal/sources/
    exit $LASTEXITCODE
}

if ($Step -eq 0) {
    Write-Host ""
    Write-Host "Usage:" -ForegroundColor Yellow
    Write-Host "  go run ./cmd/proxycheck -config $Config" -ForegroundColor DarkGray
    Write-Host "  .\scripts\test-wb-smoke.ps1 -Step 1   # config paths" -ForegroundColor DarkGray
    Write-Host "  .\scripts\test-wb-smoke.ps1 -Step 2   # session.json fresh" -ForegroundColor DarkGray
    Write-Host "  go run ./cmd/wbsession -config $Config" -ForegroundColor DarkGray
    Write-Host "  .\scripts\test-wb-smoke.ps1 -Step 3   # category (browser)" -ForegroundColor DarkGray
    Write-Host "  .\scripts\test-wb-smoke.ps1 -Step 5   # discovery API" -ForegroundColor DarkGray
    Write-Host "  .\scripts\test-wb-smoke.ps1 -E2E" -ForegroundColor DarkGray
    exit 0
}

$names = @{
    1 = "Config"
    2 = "SessionFile"
    3 = "CategoryHTTP"
    4 = "BrowserWarmup"
    5 = "DiscoveryAPI"
}
Run-Step $Step $names[$Step]
