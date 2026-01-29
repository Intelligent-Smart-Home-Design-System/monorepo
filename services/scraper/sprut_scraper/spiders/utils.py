from urllib.parse import urljoin
import re

class ProductExtractor:
    
    @staticmethod
    def extract_brand(card):
        brand = card.css('.header-pretitle ::text, h5.header-pretitle ::text, h6.header-pretitle ::text').get()
        return brand.strip() if brand else None

    @staticmethod
    def extract_title(card):
        title = card.css('h2.header-title ::text').get()
        return title.strip() if title else None

    @staticmethod
    def extract_price(card):
        price_text = card.css('h2.m-0 ::text').get()
        if not price_text:
            return None, None
        price_text_clean = price_text.strip()
        try:
            price_num = float(re.sub(r'[^\d.]', '', price_text_clean.replace(',', '.').replace(' ', '')))
        except (ValueError, TypeError):
            price_num = None
        return price_num, price_text_clean

    @staticmethod
    def extract_model(card):
        info_row = card.xpath('.//div[contains(@class, "info")]//div[contains(., "Модель:")]')
        if info_row:
            full_text = info_row.css('::text').getall()
            model_text = ''.join(full_text).replace('Модель:', '').strip()
            return model_text if model_text else None
        return None

    @staticmethod
    def extract_info_field(card, field_name):
        xpath_query = f'.//div[contains(@class, "info")]//div[contains(., "{field_name}:")]'
        info_row = card.xpath(xpath_query)
        if info_row:
            link_text = info_row.css('a ::text').get()
            if link_text:
                return link_text.strip()
            all_text = info_row.css('::text').getall()
            if len(all_text) > 1:
                return all_text[-1].strip()
        return None

    @staticmethod
    def extract_image_url(card, response):
        # Доделаю, пока не выходит вытащить url
        img_in_image = card.css('.image img[src*=".jpeg"]')
        if img_in_image:
            src = img_in_image.attrib.get('src')
            if src and not src.startswith('data:'):
                return urljoin(response.url, src)
        
        return None

    @staticmethod
    def extract_product_url(card, response):
        url = card.css('h2.header-title a::attr(href), .image a::attr(href)').get()
        return urljoin(response.url, url) if url else None

    @staticmethod
    def extract_owners_count(card):
        owners_text = card.css('.favorite-have a ::text').get() or ''
        numbers = re.findall(r'\d+', owners_text)
        return int(numbers[0]) if numbers else None

    @staticmethod
    def extract_counts(card):
        reviews_text = card.css('a.btn-sm .fa-smile').xpath('../text()').get() or ''
        manuals_text = card.css('a.btn-sm .fa-book-open').xpath('../text()').get() or ''
        
        reviews_num = re.findall(r'\d+', reviews_text)
        manuals_num = re.findall(r'\d+', manuals_text)
        
        return (
            int(reviews_num[0]) if reviews_num else None,
            int(manuals_num[0]) if manuals_num else None
        )

    @staticmethod
    def extract_rating(card):
        stars = card.css('.vue-star-rating-star')
        if not stars:
            return None
        filled_stars = 0
        for star in stars:
            star_html = star.get()
            if 'stop-color="#f6c343"' in star_html:
                if 'offset="100%"' not in star_html:
                    filled_stars += 1
                else:
                    filled_stars += 1
        
        return min(filled_stars, 5)
