from extractor.domain.models import ListingSnapshot
from extractor.adapters.outlines_extractor import OutlinesExtractor
from extractor.evaluation.metrics import ListingResult


def is_perfect(expected: dict, actual: dict) -> bool:
    """All fields match"""
    for field_name, expected_val in expected.items():
        actual_val = actual.get(field_name)
        if isinstance(expected_val, list):
            if set(expected_val) != set(actual_val):
                return False
        else:
            if expected_val != actual_val:
                return False
    return True


async def evaluate_listing(
    extractor: OutlinesExtractor,
    listing_id: int,
    listing_raw: dict,
    ground_truth: dict,
) -> ListingResult:
    expected_type = ground_truth["device_type"]
    expected_attrs = ground_truth["device_attributes"]

    try:
        listing = ListingSnapshot(**listing_raw)
        detected = await extractor.detect_device_type(listing)
        actual_type = detected.type

        if expected_type != actual_type:
            return ListingResult(
                listing_id=listing_id,
                listing_name=listing.name,
                expected_type=expected_type,
                actual_type=actual_type,
                type_correct=False,
                perfect=False,
                error=None,
                expected_output=expected_attrs,
                actual_output={},
            )
        
        extracted_attrs = await extractor.extract(listing, actual_type)

        return ListingResult(
            listing_id=listing_id,
            listing_name=listing.name,
            expected_type=expected_type,
            actual_type=actual_type,
            type_correct=True,
            perfect=is_perfect(expected_attrs, extracted_attrs),
            error=None,
            expected_output=expected_attrs,
            actual_output=extracted_attrs,
        )

    except Exception as e:
        return ListingResult(
            listing_id=listing_id,
            listing_name="error",
            expected_type=expected_type,
            actual_type="error",
            type_correct=False,
            perfect=False,
            error=str(e),
            expected_output=expected_attrs,
            actual_output={},
        )
