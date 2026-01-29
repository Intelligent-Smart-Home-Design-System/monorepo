import scrapy
from itemloaders.processors import TakeFirst

class SprutProductItem(scrapy.Item):
    title = scrapy.Field(output_processor=TakeFirst())
    brand = scrapy.Field(output_processor=TakeFirst())
    price = scrapy.Field(output_processor=TakeFirst())
    price_text = scrapy.Field(output_processor=TakeFirst())
    model = scrapy.Field(output_processor=TakeFirst())
    device_type = scrapy.Field(output_processor=TakeFirst())
    platform = scrapy.Field(output_processor=TakeFirst())
    protocol = scrapy.Field(output_processor=TakeFirst())

    reviews_count = scrapy.Field(output_processor=TakeFirst())
    manuals_count = scrapy.Field(output_processor=TakeFirst())
    owners_count = scrapy.Field(output_processor=TakeFirst())
    rating = scrapy.Field(output_processor=TakeFirst())

    product_url = scrapy.Field(output_processor=TakeFirst())
    image_url = scrapy.Field(output_processor=TakeFirst())

    scraped_at = scrapy.Field(output_processor=TakeFirst())