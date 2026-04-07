import scrapy
from datetime import datetime
from urllib.parse import urljoin
from sprut_scraper.items import SprutProductItem
from .utils import ProductExtractor

class SprutSpider(scrapy.Spider):
    name = 'sprut'
    allowed_domains = ['sprut.ai']
    start_urls = ['https://sprut.ai/catalog']
    
    custom_settings = {
        'CONCURRENT_REQUESTS': 1,
        'DOWNLOAD_DELAY': 2,
        'FEED_EXPORT_FIELDS': [
            'title', 'brand', 'price', 'price_text', 'model', 
            'device_type', 'platform', 'protocol', 'reviews_count',
            'manuals_count', 'owners_count', 'rating', 'product_url',
            'image_url', 'scraped_at'
        ]
    }
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.extractor = ProductExtractor()
    
    def parse(self, response):
        self.logger.info(f'Parsing catalog page: {response.url}')
        product_cards = response.css('.card.catalog-item.list-complete-item')
        
        for card in product_cards:
            product_item = self.parse_product_card(card, response)
            if product_item:
                yield product_item

        next_page = response.css('a:contains("›")::attr(href)').get()
        if not next_page:
            from urllib.parse import urlparse, parse_qs, urlencode, urlunparse
            parsed = urlparse(response.url)
            query = parse_qs(parsed.query)
            query['page'] = int(query.get('page', [1])[0]) + 1
            next_page = urlunparse(parsed._replace(query=urlencode(query, doseq=True)))
        
        if next_page:
            yield response.follow(next_page, self.parse)
    
    def parse_product_card(self, card, response):
        try:
            item = SprutProductItem()
            item['title'] = self.extractor.extract_title(card)
            item['brand'] = self.extractor.extract_brand(card)

            price_num, price_text = self.extractor.extract_price(card)
            item['price'] = price_num
            item['price_text'] = price_text

            item['model'] = self.extractor.extract_model(card)
            item['device_type'] = self.extractor.extract_info_field(card, 'Тип устройства')
            item['platform'] = self.extractor.extract_info_field(card, 'Платформа')
            item['protocol'] = self.extractor.extract_info_field(card, 'Протокол')
            
            item['owners_count'] = self.extractor.extract_owners_count(card)
            reviews_count, manuals_count = self.extractor.extract_counts(card)
            item['reviews_count'] = reviews_count
            item['manuals_count'] = manuals_count
            item['rating'] = self.extractor.extract_rating(card)
            
            item['image_url'] = self.extractor.extract_image_url(card, response)
            item['product_url'] = self.extractor.extract_product_url(card, response)
            item['scraped_at'] = datetime.now().isoformat()
            
            if not item['title']:
                self.logger.warning(f'Missing title for product on page {response.url}')
                return None
                
            return item
            
        except Exception as e:
            self.logger.error(f'Error parsing product card: {e}')
            return None
