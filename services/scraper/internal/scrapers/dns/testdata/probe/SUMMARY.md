# DNS fetch probe report

Generated: 2026-06-28T15:24:22+03:00

# DNS probe: https://www.dns-shop.ru/search/?q=%D1%83%D0%BC%D0%BD%D1%8B%D0%B9+%D0%B4%D0%BE%D0%BC&page=1

| Field | Value |
|-------|-------|
| Status | 200 |
| Content-Type |  |
| Body size | 73134 bytes |
| Title | Умный дом купить в интернет-магазине DNS. Умный дом цены, большой каталог, новинки. |
| **Case** | **ssr_hybrid_js_prices** |
| Reason | 17 product link(s) in HTML; prices/availability likely loaded by JS (AjaxState) |

## Signals

- has_lazy_prices: false
- has_nuxt_state: false
- has_react_root: false
- has_json_ld: true
- has_captcha_page: false
- has_search_results: false
- has_ajax_state: true
- has_catalog_links: true
- has_product_links: false
- has_product_microdata: false
- has_access_denied: false
- has_price_rub: false

## Sample /catalog/ paths

- `/catalog/17aa343116404e77/umnyj-dom`
- `/catalog/9bbe8fe270e7c3ae/umnaa-tehnika`
- `/catalog/7deac3d82c89d034/ekosistemy-umnogo-doma`
- `/catalog/e424bddb99192176/upravlenie-scenariami-umnogo-doma`
- `/catalog/f2e8296973da4e77/centry-upravlenia-umnym-domom-haby`
- `/catalog/recipe/5c5a3cada3a0b8ff`
- `/catalog/recipe/dbfeaee02d85fa4b`
- `/catalog/recipe/146ab0c73fadaca2`
- `/catalog/b8da90155a1073dc/umnye-pulty`
- `/catalog/recipe/f7454da17358df00`
- `/catalog/recipe/39e2e0afdb01d0d1`
- `/catalog/recipe/e65abb811276b51a`
- `/catalog/abfdb39ce97757cb/ustrojstva-upravlenia-umnym-domom`
- `/catalog/59bc9aedc20db24d/umnaa-audiotehnika`
- `/catalog/4c317e80374078d9/umnyj-svet`
- `/catalog/d58d4f35254bb166/umnyj-dom-andeks`
- `/catalog/17ba8ffd1768a6fd/umnye-datciki`

## Recommendations

- Product cards are in SSR HTML — discovery/listing names and URLs can be parsed without browser.
- Prices and buy buttons load via AjaxState/JS — either call internal API or use headless browser for price.
- Fix discovery regex: DNS product URLs are /product/{id}/{slug}/ not /catalog/...

## Body preview (whitespace collapsed)

```
<!DOCTYPE html><html lang="ru"><head><meta http-equiv="X-UA-Compatible" content="IE=edge" /><meta charset="UTF-8" /><meta name="format-detection" content="telephone=no"><meta name="csrf-param" content="_csrf"><meta name="csrf-token" content="l6m1P58J_ArFIm2c37dlC-l5sEjnniDxDF91VLoFmtj7_cdlxiSGXoZOG_SM2yw-jTXlAr79codYbz0z8HH-tg=="><link rel="shortcut icon" href="https://a.dns-shop.ru/static/06/1say95p/static/favicon.png" type="image/png" /><title>Умный дом купить в интернет-магазине DNS. Умный дом цены, большой каталог, новинки.</title><link rel="manifest" href="https://a.dns-shop.ru/web-files/manifest/manifest.json" /><meta name="theme-color" content="#F6F6F6" /><meta name="application-name" content="DNS" /><meta name="mobile-web-app-capable" content="yes" /><meta name="viewport" content="width=device-width, initial-scale=1" /><link rel="icon" type="image/png" sizes="16x16" href="https://a.dns-shop.ru/web-files/manifest/favicon-16x16.png" /><link rel="icon" type="image/png" sizes="24x24" href="https://a.dns-shop.ru/web-files/manifest/favicon-24x24.png" /><link rel="icon" type="image/png" sizes="32x32" href="https://a.dns-shop.ru/web-files/manifest/favicon-32x32.png" /><link rel="icon" type="image/png" sizes="48x48" href="https://a.dns-shop.ru/web-files/manifest/favicon-48x48.png" /><link rel="icon" type="image/png" sizes="96x96" href="https://a.dns-shop.ru/web-files/manifest/favicon-96x96.png" /><link rel="icon" type="image/png" sizes="192x192" href="https://a.dns-shop.ru/web-files/manifest/android-icon-192x192.png" /><meta name="apple-mobile-web-app-capable" content="yes" /><meta name="apple-mobile-web-app-title" content="DNS" /><meta name="apple-mobile-web-app-status-bar-style" content="default" /><link rel="apple-touch-icon" sizes="48x48" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-48x48.png" /><link rel="apple-touch-icon" sizes="57x57" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-57x57.png" /><link rel="apple-touch-icon" sizes="60x60" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-60x60.png" /><link rel="apple-touch-icon" sizes="72x72" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-72x72.png" /><link rel="apple-touch-icon" sizes="76x76" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-76x76.png" /><link rel="apple-touch-icon" sizes="96x96" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-96x96.png" /><link rel="apple-touch-…
```

---

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

---

# DNS probe: https://www.dns-shop.ru/product/420b8a9ab11ad0a4/datcik-moes-zigbee-flood-sensor-water-leakage-detector/

| Field | Value |
|-------|-------|
| Status | 200 |
| Content-Type |  |
| Body size | 80074 bytes |
| Title | Купить Датчик MOES ZigBee Flood Sensor Water Leakage Detector в интернет-магазине DNS. Характеристики, цена MOES ZigBee Flood Sensor Water Leakage Detector | 5608711. |
| **Case** | **ssr_hybrid_js_prices** |
| Reason | 12 product link(s) in HTML; prices/availability likely loaded by JS (AjaxState) |

## Signals

- has_react_root: false
- has_json_ld: false
- has_product_microdata: false
- has_captcha_page: false
- has_access_denied: false
- has_search_results: false
- has_ajax_state: true
- has_catalog_links: true
- has_product_links: true
- has_price_rub: false
- has_lazy_prices: true
- has_nuxt_state: false

## Sample /product/ paths

- `/product/420b8a9ab11ad0a4/datcik-moes-zigbee-flood-sensor-water-leakage-detector/`

## Sample /catalog/ paths

- `/catalog/17aa343116404e77/umnyj-dom`
- `/catalog/9bbe8fe270e7c3ae/umnaa-tehnika`
- `/catalog/195a8a0d7cee53b3/umnye-datciki`
- `/catalog/17a9e64016404e77/umnye-datciki`
- `/catalog/product/get-rating-img`
- `/catalog/product/product-social-activity-page`
- `/catalog/product/product-reviews`
- `/catalog/product/get-product-accessories-slider`
- `/catalog/product/get-product-buy-together-slider`
- `/catalog/product/get-product-analogs-slider`
- `/catalog/product/get-product-similar-slider`

## Recommendations

- Product cards are in SSR HTML — discovery/listing names and URLs can be parsed without browser.
- Prices and buy buttons load via AjaxState/JS — either call internal API or use headless browser for price.
- Fix discovery regex: DNS product URLs are /product/{id}/{slug}/ not /catalog/...

## Body preview (whitespace collapsed)

```
<!DOCTYPE html><html lang="ru"><head><meta http-equiv="X-UA-Compatible" content="IE=edge" /><meta charset="UTF-8" /><meta name="csrf-param" content="_csrf"><meta name="csrf-token" content="ufkyvT2s-aFS1DRGoA7bBjM2CocZRuGh0Sd8r-mux67Kml31Vv6-lxTnWXTZSJJtYmNHzUMyrNmFcRTK29yrzQ=="><link rel="shortcut icon" href="https://a.dns-shop.ru/static/06/1say95p/static/favicon.png" type="image/png" /><title>Купить Датчик MOES ZigBee Flood Sensor Water Leakage Detector в интернет-магазине DNS. Характеристики, цена MOES ZigBee Flood Sensor Water Leakage Detector | 5608711.</title><link rel="manifest" href="https://a.dns-shop.ru/web-files/manifest/manifest.json" /><meta name="theme-color" content="#F6F6F6" /><meta name="application-name" content="DNS" /><meta name="mobile-web-app-capable" content="yes" /><meta name="viewport" content="width=device-width, initial-scale=1" /><link rel="icon" type="image/png" sizes="16x16" href="https://a.dns-shop.ru/web-files/manifest/favicon-16x16.png" /><link rel="icon" type="image/png" sizes="24x24" href="https://a.dns-shop.ru/web-files/manifest/favicon-24x24.png" /><link rel="icon" type="image/png" sizes="32x32" href="https://a.dns-shop.ru/web-files/manifest/favicon-32x32.png" /><link rel="icon" type="image/png" sizes="48x48" href="https://a.dns-shop.ru/web-files/manifest/favicon-48x48.png" /><link rel="icon" type="image/png" sizes="96x96" href="https://a.dns-shop.ru/web-files/manifest/favicon-96x96.png" /><link rel="icon" type="image/png" sizes="192x192" href="https://a.dns-shop.ru/web-files/manifest/android-icon-192x192.png" /><meta name="apple-mobile-web-app-capable" content="yes" /><meta name="apple-mobile-web-app-title" content="DNS" /><meta name="apple-mobile-web-app-status-bar-style" content="default" /><link rel="apple-touch-icon" sizes="48x48" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-48x48.png" /><link rel="apple-touch-icon" sizes="57x57" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-57x57.png" /><link rel="apple-touch-icon" sizes="60x60" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-60x60.png" /><link rel="apple-touch-icon" sizes="72x72" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-72x72.png" /><link rel="apple-touch-icon" sizes="76x76" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-76x76.png" /><link rel="apple-touch-icon" sizes="96x96" href="https://a.dns-shop.ru/web-files/manifest/apple-icon-96x96.png" /><link rel=…
```

---

