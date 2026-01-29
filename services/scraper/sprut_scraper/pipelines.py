from datetime import datetime
from scrapy.exceptions import DropItem

class SprutScraperPipeline:
    def process_item(self, item, spider):
        if not item.get('title'):
            raise DropItem(f"Missing title in item: {item}")

        if item.get('price_text'):
            item['price_text'] = item['price_text'].replace('&nbsp;', ' ').strip()

        item['scraped_at'] = datetime.now().isoformat()
        
        return item
