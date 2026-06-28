# DNS probe: https://www.dns-shop.ru/search/?q=zigbee&page=1

| Field | Value |
|-------|-------|
| Status | 200 |
| Content-Type |  |
| Body size | 195731 bytes |
| Title | zigbee - DNS – интернет магазин цифровой и бытовой техники по доступным ценам. |
| **Case** | **ssr_hybrid_js_prices** |
| Reason | 7 product link(s) in HTML; prices/availability likely loaded by JS (AjaxState) |

## Signals

- has_catalog_links: true
- has_product_links: true
- has_react_root: false
- has_product_microdata: false
- has_captcha_page: false
- has_access_denied: false
- has_price_rub: false
- has_search_results: true
- has_nuxt_state: false
- has_json_ld: false
- has_ajax_state: true
- has_lazy_prices: true

## Sample /product/ paths

- `/product/420b8a9ab11ad0a4/datcik-moes-zigbee-flood-sensor-water-leakage-detector/`
- `/product/3c0deaefb11ad0a4/datcik-moes-zigbee-window-door-alarm-sensor/`
- `/product/8fa02ec003a5d0a4/datcik-moes-zigbee-temperature-and-humidity-sensor/`
- `/product/83af676e03a5d0a4/datcik-moes-zigbee-smoke-detector/`

## Sample /catalog/ paths

- `/catalog/17a9e64016404e77/umnye-datciki`
- `/catalog/category/log-filters-statistics`
- `/catalog/product/get-option-description`

## Recommendations

- Product cards are in SSR HTML — discovery/listing names and URLs can be parsed without browser.
- Prices and buy buttons load via AjaxState/JS — either call internal API or use headless browser for price.
- Fix discovery regex: DNS product URLs are /product/{id}/{slug}/ not /catalog/...

## Body preview (whitespace collapsed)

```
<!DOCTYPE html><html lang="ru"><head><meta http-equiv="X-UA-Compatible" content="IE=edge" /><meta charset="UTF-8" /><meta name="format-detection" content="telephone=no"><meta name="csrf-param" content="_csrf"><meta name="csrf-token" content="Sh5UEppO0QOsTNU2AM3bGJbu4_wVySJ99Ruu2Wm7k08fbhpd6iaOUc8condMhr1f57jXhny7S0WHc8zrXPfVHA=="><link rel="shortcut icon" href="https://a.dns-shop.ru/static/06/1say95p/static/favicon.png" type="image/png" /><title>zigbee - DNS – интернет магазин цифровой и бытовой техники по доступным ценам.</title><link rel="manifest" href="https://a.dns-shop.ru/web-files/manifest/manifest.json" /><meta name="theme-color" content="#F6F6F6" /><meta name="application-name" content="DNS" /><meta name="mobile-web-app-capable" content="yes" /><meta name="viewport" content="width=device-width, initial-scale=1" /><link rel="icon" type="image/png" sizes="16x16" href="https://a.dns-shop.ru/web-files/manifest/favicon-16x16.png" /><link rel="icon" type="image/png" sizes="24x24" href="https://a.dns-shop.ru/web-files/manifest/favicon-24x24.png" /><link rel="icon" type="image/png" sizes="32x32" href="https://a.dns-shop.ru/web-files/manifest/favicon-32x32.png" /><link rel="icon" type="image/png" sizes="48x48" href="https://a.dns-shop.ru/web-files/manifest/favicon-48x48.png" /><link rel="icon" type="image/png" sizes="96x96" href="https://a.dns-shop.ru/web-files/manifest/favicon-96x96.png" /><link rel="icon" type="image/png" sizes="192x192" href="https://a.dns-shop.ru/web-files/manifest/android-icon-192x192.png" /><meta name="apple-mobile-web-app-capable" content="yes" /><meta name="apple-mobile-web-app-title" content="DNS" /><meta name="apple-mobile-web-app-status-bar-style" content="default" /><link rel="apple-touch-icon" sizes="48x48" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-48x48.png" /><link rel="apple-touch-icon" sizes="57x57" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-57x57.png" /><link rel="apple-touch-icon" sizes="60x60" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-60x60.png" /><link rel="apple-touch-icon" sizes="72x72" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-72x72.png" /><link rel="apple-touch-icon" sizes="76x76" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-76x76.png" /><link rel="apple-touch-icon" sizes="96x96" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-96x96.png" /><link rel="apple-touch-icon" sizes="…
```
