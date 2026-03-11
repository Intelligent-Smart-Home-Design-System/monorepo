from pydantic import BaseModel
from datetime import datetime
from typing import Any


class ListingSnapshot(BaseModel):
    id: int
    name: str 
    in_stock: bool
    text: str
    brand: str
    rating: float
    review_count: int
    price: int | None
    currency: str | None
    model_number: str | None
    category: str | None
    quantity: int | None


class ExtractionSnapshot(BaseModel):
    parsed_listing_snapshot_id: int
    brand: str
    model: str
    category: str
    category_confidence: float
    extracted_at: datetime
    llm_model: str
    device_attributes: dict[str, Any]
