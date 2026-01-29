from scrapy import signals
import random
import time


class SprutScraperDownloaderMiddleware:
    """Промежуточное ПО для обработки запросов и ответов"""
    
    @classmethod
    def from_crawler(cls, crawler):
        middleware = cls()
        crawler.signals.connect(middleware.spider_opened, signal=signals.spider_opened)
        crawler.signals.connect(middleware.spider_closed, signal=signals.spider_closed)
        return middleware
    
    def spider_opened(self, spider):
        spider.logger.info('Spider opened: %s' % spider.name)
    
    def spider_closed(self, spider):
        spider.logger.info('Spider closed: %s' % spider.name)
    
    def process_request(self, request, spider):
        request.headers['Accept'] = 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8'
        request.headers['Accept-Language'] = 'ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7'
        request.headers['Accept-Encoding'] = 'gzip, deflate, br'
        request.headers['Connection'] = 'keep-alive'
        request.headers['Upgrade-Insecure-Requests'] = '1'
        request.headers['Sec-Fetch-Dest'] = 'document'
        request.headers['Sec-Fetch-Mode'] = 'navigate'
        request.headers['Sec-Fetch-Site'] = 'none'
        request.headers['Sec-Fetch-User'] = '?1'
    
        time.sleep(random.uniform(0.5, 1.5))
        
        return None
    
    def process_response(self, request, response, spider):
        if response.status != 200:
            spider.logger.warning(f'Received status {response.status} for {request.url}')
        
        if any(indicator in response.text for indicator in ['captcha', 'CAPTCHA', 'Cloudflare']):
            spider.logger.error(f'Captcha detected on {request.url}')
        
        return response
    
    def process_exception(self, request, exception, spider):
        spider.logger.error(f'Exception occurred: {exception} for {request.url}')
        return None