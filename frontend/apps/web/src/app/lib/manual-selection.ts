export const MANUAL_SELECTION_STORAGE_KEY = "planner-manual-selection";
export const MANUAL_MODE_STORAGE_KEY = "planner-selection-mode";

export type ManualSelectionItem = {
  categoryId: string;
  deviceId: number;
  listingId: number;
  name: string;
  brand: string;
  model: string;
  imageUrl?: string | null;
  quantity: number;
  unitPrice: number;
  currency: string;
  devicesPerListing: number;
};

export function loadManualSelection(): ManualSelectionItem[] {
  if (typeof window === "undefined") return [];

  try {
    const raw = localStorage.getItem(MANUAL_SELECTION_STORAGE_KEY);
    const parsed = raw ? (JSON.parse(raw) as unknown) : [];
    if (!Array.isArray(parsed)) return [];

    const byCategory = new Map<string, ManualSelectionItem>();
    parsed.forEach((value) => {
      if (!value || typeof value !== "object") return;
      const item = value as Partial<ManualSelectionItem>;
      if (
        typeof item.categoryId !== "string" ||
        !item.categoryId ||
        typeof item.deviceId !== "number" ||
        !Number.isFinite(item.deviceId) ||
        typeof item.listingId !== "number" ||
        !Number.isFinite(item.listingId) ||
        typeof item.name !== "string" ||
        !item.name ||
        typeof item.unitPrice !== "number" ||
        !Number.isFinite(item.unitPrice) ||
        item.unitPrice < 0
      ) {
        return;
      }

      byCategory.set(item.categoryId, {
        categoryId: item.categoryId,
        deviceId: item.deviceId,
        listingId: item.listingId,
        name: item.name,
        brand: typeof item.brand === "string" ? item.brand : "",
        model: typeof item.model === "string" ? item.model : "",
        imageUrl: typeof item.imageUrl === "string" ? item.imageUrl : null,
        quantity:
          typeof item.quantity === "number" && Number.isFinite(item.quantity)
            ? Math.max(1, Math.floor(item.quantity))
            : 1,
        unitPrice: item.unitPrice,
        currency: typeof item.currency === "string" && item.currency ? item.currency : "RUB",
        devicesPerListing:
          typeof item.devicesPerListing === "number" && Number.isFinite(item.devicesPerListing)
            ? Math.max(1, Math.floor(item.devicesPerListing))
            : 1,
      });
    });

    return Array.from(byCategory.values());
  } catch {
    return [];
  }
}

export function saveManualSelection(items: ManualSelectionItem[]) {
  if (typeof window === "undefined") return;

  if (items.length === 0) {
    localStorage.removeItem(MANUAL_SELECTION_STORAGE_KEY);
    return;
  }
  localStorage.setItem(MANUAL_SELECTION_STORAGE_KEY, JSON.stringify(items));
}

export function upsertManualSelection(item: ManualSelectionItem) {
  const items = loadManualSelection();
  saveManualSelection([...items.filter((current) => current.categoryId !== item.categoryId), item]);
}

export function manualSelectionCost(item: ManualSelectionItem) {
  return item.unitPrice * Math.ceil(item.quantity / Math.max(item.devicesPerListing, 1));
}
