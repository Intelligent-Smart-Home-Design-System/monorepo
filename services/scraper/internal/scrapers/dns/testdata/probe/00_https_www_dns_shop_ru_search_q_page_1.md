# DNS probe: https://www.dns-shop.ru/search/?q=умный+дом&page=1

| Field | Value |
|-------|-------|
| Status | 400 |
| Content-Type | text/html |
| Body size | 90 bytes |
| Title |  |
| **Case** | **spa_empty_shell** |
| Reason | small body (90 bytes) without catalog links |

## Signals

- has_nuxt_state: false
- has_react_root: false
- has_json_ld: false
- has_product_microdata: false
- has_access_denied: false
- has_price_rub: false
- has_search_results: false
- has_catalog_links: false
- has_captcha: false

## Recommendations

- Plain HTTP is NOT enough — page needs JavaScript rendering.
- Use headless browser (rod, like Wildberries session mining) or find XHR/API endpoints.

## Body preview (whitespace collapsed)

```
<html><body><h1>400 Bad request</h1> Your browser sent an invalid request. </body></html>
```
