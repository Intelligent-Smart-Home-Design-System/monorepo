from extractor.domain.models import ListingSnapshot
from extractor.enrichment import enrich_from_silver_layer


def _listing(**overrides) -> ListingSnapshot:
    base = {
        "id": 1,
        "name": "Temperature and Humidity Sensor T1",
        "in_stock": True,
        "text": "Протокол: Zigbee",
        "brand": "Aqara",
        "rating": 5.0,
        "review_count": 2,
        "price": None,
        "currency": None,
        "model_number": "WSDCGQ12LM",
        "category": "Датчики",
        "quantity": None,
        "source_name": "sprut",
        "extractor_version": "sprut_listing_v2",
    }
    base.update(overrides)
    return ListingSnapshot(**base)


class TestEnrichFromSilverLayer:
    def test_fills_missing_brand_model_and_name(self):
        brand, model, attrs = enrich_from_silver_layer(
            _listing(),
            {"type": "temperature_sensor", "protocol": ["zigbee"]},
        )

        assert brand == "aqara"
        assert model == "WSDCGQ12LM"
        assert attrs["brand"] == "aqara"
        assert attrs["model"] == "WSDCGQ12LM"
        assert attrs["name"] == "Temperature and Humidity Sensor T1"

    def test_keeps_llm_values_when_present(self):
        brand, model, attrs = enrich_from_silver_layer(
            _listing(),
            {
                "brand": "aqara",
                "model": "CUSTOM-MODEL",
                "name": "LLM Name",
                "type": "temperature_sensor",
            },
        )

        assert brand == "aqara"
        assert model == "CUSTOM-MODEL"
        assert attrs["model"] == "CUSTOM-MODEL"
        assert attrs["name"] == "LLM Name"

    def test_model_unknown_when_everything_missing(self):
        listing = _listing(model_number=None, brand="")
        brand, model, attrs = enrich_from_silver_layer(listing, {"type": "unknown"})

        assert brand == ""
        assert model == "unknown"
        assert "model" not in attrs
