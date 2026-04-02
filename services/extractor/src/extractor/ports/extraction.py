from typing import Any, Protocol
from extractor.domain.models import DetectedDeviceType, ListingSnapshot

class Extractor(Protocol):
    """
    Gets enriched information about the smart home device listing from a parsed listing snapshot.
    """

    async def detect_device_type(listing: ListingSnapshot) -> DetectedDeviceType:
        """
        Detects the device type of a listing, giving type (from a set of types given to the implementation)
        and confidence score (0.0 - 1.0)
        """
        ...

    async def extract(listing: ListingSnapshot, as_type: str) -> dict[str, Any]:
        """
        Extracts information about the smart home device from a listing, assuming the listing has device type 'as_type'.
        The output dict is guaranteed to conform to the schema of 'as_type', where all fields are made required and nullable.
        'null' for a field means that the value could not be found in the listing.
        """
        ...
