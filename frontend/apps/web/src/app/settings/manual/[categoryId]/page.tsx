"use client";

/* eslint-disable react-hooks/set-state-in-effect */

import { useEffect, useState, type ReactNode } from "react";
import { useParams, useRouter } from "next/navigation";
import ArrowBackRoundedIcon from "@mui/icons-material/ArrowBackRounded";
import CheckRoundedIcon from "@mui/icons-material/CheckRounded";
import DevicesOtherRoundedIcon from "@mui/icons-material/DevicesOtherRounded";
import FilterAltOffRoundedIcon from "@mui/icons-material/FilterAltOffRounded";
import SearchRoundedIcon from "@mui/icons-material/SearchRounded";
import StarRoundedIcon from "@mui/icons-material/StarRounded";
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  InputAdornment,
  MenuItem,
  Pagination,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import { api } from "../../../lib/api";
import { useAuth } from "../../../lib/auth-context";
import {
  loadManualSelection,
  MANUAL_MODE_STORAGE_KEY,
  manualSelectionCost,
  type ManualSelectionItem,
  upsertManualSelection,
} from "../../../lib/manual-selection";
import type {
  ApiCatalogProduct,
  ApiCatalogProductFilters,
  ApiCatalogProductsResponse,
  ApiCatalogProductsQuery,
} from "../../../lib/types";

const PAGE_SIZE = 20;
const EMPTY_FILTERS: ApiCatalogProductFilters = {
  brands: [],
  ecosystems: [],
  protocols: [],
  min_price: 0,
  max_price: 0,
};
const LIGHT_FIELD_SX = {
  "& .MuiOutlinedInput-root": {
    backgroundColor: "#fff",
    color: "#111827",
  },
  "& .MuiInputLabel-root": {
    color: "#4b5563",
  },
  "& .MuiInputLabel-root.Mui-focused": {
    color: "#2563eb",
  },
  "& .MuiSelect-icon": {
    color: "#4b5563",
  },
  "& .MuiInputBase-input::placeholder": {
    color: "#6b7280",
    opacity: 1,
  },
} as const;

type CatalogSort = NonNullable<ApiCatalogProductsQuery["sort"]>;

export default function ManualCategoryPage() {
  const auth = useAuth();
  const router = useRouter();
  const params = useParams<{ categoryId: string }>();
  const categoryId = params.categoryId;

  const [categoryName, setCategoryName] = useState(() => humanizeCategory(categoryId));
  const [response, setResponse] = useState<ApiCatalogProductsResponse | null>(null);
  const [filters, setFilters] = useState<ApiCatalogProductFilters>(EMPTY_FILTERS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [reloadKey, setReloadKey] = useState(0);

  const [searchInput, setSearchInput] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [brand, setBrand] = useState("");
  const [ecosystem, setEcosystem] = useState("");
  const [protocol, setProtocol] = useState("");
  const [minPriceInput, setMinPriceInput] = useState("");
  const [maxPriceInput, setMaxPriceInput] = useState("");
  const [sort, setSort] = useState<CatalogSort>("price_asc");
  const [page, setPage] = useState(1);
  const [selectedItems, setSelectedItems] = useState<ManualSelectionItem[]>([]);
  const [budget, setBudget] = useState<number | null>(null);

  const selectedItem = selectedItems.find((item) => item.categoryId === categoryId);
  const otherItemsTotal = selectedItems
    .filter((item) => item.categoryId !== categoryId)
    .reduce((total, item) => total + manualSelectionCost(item), 0);
  const selectedQuantity = selectedItem?.quantity ?? 1;
  const availableForCategory = budget === null ? null : budget - otherItemsTotal;

  const minPrice = optionalPrice(minPriceInput);
  const maxPrice = optionalPrice(maxPriceInput);
  const invalidPriceRange = minPrice !== undefined && maxPrice !== undefined && minPrice > maxPrice;

  useEffect(() => {
    localStorage.setItem(MANUAL_MODE_STORAGE_KEY, "manual");
    setSelectedItems(loadManualSelection());
    setBudget(readStoredBudget());
  }, []);

  useEffect(() => {
    if (!auth.isAuthenticated) return;

    let active = true;
    api
      .listCatalogCategories()
      .then((categories) => {
        if (!active) return;
        const category = categories.find((item) => item.id === categoryId);
        if (category) setCategoryName(category.name);
      })
      .catch(() => {
        // The products endpoint remains usable when category metadata is unavailable.
      });

    return () => {
      active = false;
    };
  }, [auth.isAuthenticated, categoryId]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setSearchQuery(searchInput.trim());
      setPage(1);
    }, 350);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    if (!auth.isAuthenticated || invalidPriceRange) {
      if (invalidPriceRange) setLoading(false);
      return;
    }

    let active = true;
    setLoading(true);
    setError("");

    api
      .listCatalogProducts({
        device_type: categoryId,
        q: searchQuery || undefined,
        brand: brand || undefined,
        ecosystem: ecosystem || undefined,
        protocol: protocol || undefined,
        min_price: minPrice,
        max_price: maxPrice,
        sort,
        page,
        page_size: PAGE_SIZE,
      })
      .then((result) => {
        if (!active) return;
        setResponse(result);
        setFilters(result.filters);
      })
      .catch((err: unknown) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Не удалось загрузить модели устройств.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [
    auth.isAuthenticated,
    brand,
    categoryId,
    ecosystem,
    invalidPriceRange,
    maxPrice,
    minPrice,
    page,
    protocol,
    reloadKey,
    searchQuery,
    sort,
  ]);

  const resetFilters = () => {
    setSearchInput("");
    setSearchQuery("");
    setBrand("");
    setEcosystem("");
    setProtocol("");
    setMinPriceInput("");
    setMaxPriceInput("");
    setSort("price_asc");
    setPage(1);
  };

  const chooseProduct = (product: ApiCatalogProduct) => {
    upsertManualSelection({
      categoryId,
      deviceId: product.device_id,
      listingId: product.listing_id,
      name: product.name,
      brand: product.brand,
      model: product.model,
      imageUrl: product.image_url,
      quantity: selectedQuantity,
      unitPrice: product.price,
      currency: product.currency || "RUB",
      devicesPerListing: product.devices_per_listing,
    });
    router.push("/settings");
  };

  const hasActiveFilters = Boolean(
    searchInput || brand || ecosystem || protocol || minPriceInput || maxPriceInput || sort !== "price_asc"
  );

  if (auth.loading) {
    return <CenteredProgress />;
  }

  if (!auth.isAuthenticated) {
    return (
      <PageFrame>
        <Alert
          severity="warning"
          action={
            <Button color="inherit" size="small" onClick={() => router.push(`/login?next=/settings/manual/${categoryId}`)}>
              Войти
            </Button>
          }
        >
          Для ручного подбора нужно войти в аккаунт.
        </Alert>
      </PageFrame>
    );
  }

  return (
    <PageFrame>
      <Stack spacing={3}>
        <Box>
          <Button
            startIcon={<ArrowBackRoundedIcon />}
            onClick={() => router.push("/settings")}
            sx={{
              mb: 1.5,
              color: "#fff",
              "&:hover": { backgroundColor: "rgba(255,255,255,0.10)" },
            }}
          >
            К списку устройств
          </Button>
          <Stack
            direction={{ xs: "column", md: "row" }}
            spacing={2}
            justifyContent="space-between"
            alignItems={{ xs: "stretch", md: "flex-end" }}
          >
            <Box>
              <Typography variant="h4" sx={{ fontWeight: 850, mb: 0.75 }}>
                {categoryName}
              </Typography>
              <Typography sx={{ color: "rgba(255,255,255,0.82)" }}>
                Найдите подходящую модель и добавьте её в конфигурацию.
              </Typography>
            </Box>
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
              {budget !== null && (
                <Chip
                  label={`Бюджет: ${formatPrice(budget)} ₽`}
                  variant="outlined"
                  sx={{ color: "#fff", borderColor: "rgba(255,255,255,0.72)" }}
                />
              )}
              {availableForCategory !== null && (
                <Chip
                  label={`Доступно: ${formatPrice(Math.max(0, availableForCategory))} ₽`}
                  color={availableForCategory < 0 ? "error" : "default"}
                  variant="outlined"
                  sx={
                    availableForCategory < 0
                      ? undefined
                      : { color: "#fff", borderColor: "rgba(255,255,255,0.72)" }
                  }
                />
              )}
            </Stack>
          </Stack>
        </Box>

        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: { xs: "minmax(0, 1fr)", lg: "240px minmax(0, 1fr)" },
            gap: { xs: 2.5, lg: 3 },
            alignItems: "start",
          }}
        >
          <Box
            component="aside"
            aria-label="Фильтры каталога"
            sx={{
              border: "1px solid rgba(148,163,184,0.32)",
              borderRadius: 2,
              p: 2,
              backgroundColor: "#fff",
              color: "#111827",
            }}
          >
            <Stack spacing={1.75}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography sx={{ fontWeight: 800, color: "#111827" }}>Фильтры</Typography>
                <Button
                  size="small"
                  startIcon={<FilterAltOffRoundedIcon />}
                  disabled={!hasActiveFilters}
                  onClick={resetFilters}
                  sx={{ color: "#4b5563" }}
                >
                  Сбросить
                </Button>
              </Stack>

              <TextField
                label="Бренд"
                select
                size="small"
                value={brand}
                onChange={(event) => {
                  setBrand(event.target.value);
                  setPage(1);
                }}
                sx={LIGHT_FIELD_SX}
              >
                <MenuItem value="">Все бренды</MenuItem>
                {filters.brands.map((item) => (
                  <MenuItem key={item} value={item}>
                    {item}
                  </MenuItem>
                ))}
              </TextField>

              <TextField
                label="Экосистема"
                select
                size="small"
                value={ecosystem}
                onChange={(event) => {
                  setEcosystem(event.target.value);
                  setPage(1);
                }}
                sx={LIGHT_FIELD_SX}
              >
                <MenuItem value="">Все экосистемы</MenuItem>
                {filters.ecosystems.map((item) => (
                  <MenuItem key={item} value={item}>
                    {humanizeCategory(item)}
                  </MenuItem>
                ))}
              </TextField>

              <TextField
                label="Протокол"
                select
                size="small"
                value={protocol}
                onChange={(event) => {
                  setProtocol(event.target.value);
                  setPage(1);
                }}
                sx={LIGHT_FIELD_SX}
              >
                <MenuItem value="">Все протоколы</MenuItem>
                {filters.protocols.map((item) => (
                  <MenuItem key={item} value={item}>
                    {item.toUpperCase()}
                  </MenuItem>
                ))}
              </TextField>

              <Stack direction={{ xs: "row", lg: "column" }} spacing={1}>
                <TextField
                  label="Цена от"
                  size="small"
                  value={minPriceInput}
                  onChange={(event) => {
                    setMinPriceInput(event.target.value.replace(/[^\d]/g, ""));
                    setPage(1);
                  }}
                  inputMode="numeric"
                  sx={{ ...LIGHT_FIELD_SX, flex: 1 }}
                />
                <TextField
                  label="Цена до"
                  size="small"
                  value={maxPriceInput}
                  onChange={(event) => {
                    setMaxPriceInput(event.target.value.replace(/[^\d]/g, ""));
                    setPage(1);
                  }}
                  inputMode="numeric"
                  error={invalidPriceRange}
                  helperText={invalidPriceRange ? "Меньше цены «от»" : " "}
                  sx={{
                    ...LIGHT_FIELD_SX,
                    flex: 1,
                    "& .MuiFormHelperText-root": {
                      color: invalidPriceRange ? "#d32f2f" : "#4b5563",
                    },
                  }}
                />
              </Stack>
            </Stack>
          </Box>

          <Box component="section" aria-label="Модели устройств" sx={{ minWidth: 0 }}>
            <Stack
              direction={{ xs: "column", sm: "row" }}
              spacing={1.5}
              justifyContent="space-between"
              sx={{ mb: 2 }}
            >
              <TextField
                placeholder="Поиск по названию, бренду или модели"
                value={searchInput}
                onChange={(event) => setSearchInput(event.target.value)}
                size="small"
                fullWidth
                slotProps={{
                  input: {
                    startAdornment: (
                      <InputAdornment position="start">
                        <SearchRoundedIcon />
                      </InputAdornment>
                    ),
                  },
                }}
                sx={LIGHT_FIELD_SX}
              />
              <TextField
                select
                size="small"
                value={sort}
                onChange={(event) => {
                  setSort(event.target.value as CatalogSort);
                  setPage(1);
                }}
                sx={{
                  ...LIGHT_FIELD_SX,
                  width: { xs: "100%", sm: 220 },
                  flex: "0 0 auto",
                }}
                slotProps={{ select: { "aria-label": "Сортировка" } }}
              >
                <MenuItem value="price_asc">Сначала дешевле</MenuItem>
                <MenuItem value="price_desc">Сначала дороже</MenuItem>
                <MenuItem value="rating_desc">По рейтингу</MenuItem>
                <MenuItem value="name_asc">По названию</MenuItem>
              </TextField>
            </Stack>

            {invalidPriceRange && (
              <Alert severity="warning" sx={{ mb: 2 }}>
                Максимальная цена должна быть не меньше минимальной.
              </Alert>
            )}
            {error && (
              <Alert
                severity="error"
                sx={{ mb: 2 }}
                action={
                  <Button color="inherit" size="small" onClick={() => setReloadKey((value) => value + 1)}>
                    Повторить
                  </Button>
                }
              >
                {error}
              </Alert>
            )}

            {loading ? (
              <Box sx={{ minHeight: 360, display: "grid", placeItems: "center" }}>
                <CircularProgress />
              </Box>
            ) : response?.items.length ? (
              <>
                <Typography variant="body2" sx={{ mb: 1.25, color: "rgba(255,255,255,0.82)" }}>
                  Найдено моделей: {response.total}
                </Typography>
                <Box
                  sx={{
                    border: "1px solid rgba(148,163,184,0.32)",
                    borderRadius: 2,
                    overflow: "hidden",
                    backgroundColor: "#fff",
                  }}
                >
                  {response.items.map((product, index) => (
                    <ProductRow
                      key={product.device_id}
                      product={product}
                      first={index === 0}
                      selected={selectedItem?.deviceId === product.device_id}
                      quantity={selectedQuantity}
                      availableForCategory={availableForCategory}
                      onChoose={() => chooseProduct(product)}
                    />
                  ))}
                </Box>

                {response.total_pages > 1 && (
                  <Box sx={{ mt: 3, display: "flex", justifyContent: "center" }}>
                    <Pagination
                      page={response.page}
                      count={response.total_pages}
                      onChange={(_, nextPage) => {
                        setPage(nextPage);
                        window.scrollTo({ top: 0, behavior: "smooth" });
                      }}
                      color="primary"
                      siblingCount={0}
                    />
                  </Box>
                )}
              </>
            ) : !invalidPriceRange && !error ? (
              <Box
                sx={{
                  minHeight: 280,
                  display: "grid",
                  placeItems: "center",
                  textAlign: "center",
                  border: "1px solid rgba(148,163,184,0.32)",
                  borderRadius: 2,
                  px: 3,
                }}
              >
                <Box>
                  <DevicesOtherRoundedIcon sx={{ fontSize: 40, color: "text.secondary", mb: 1 }} />
                  <Typography sx={{ fontWeight: 800, mb: 0.5 }}>Модели не найдены</Typography>
                  <Typography variant="body2" color="text.secondary">
                    Измените запрос или сбросьте фильтры.
                  </Typography>
                </Box>
              </Box>
            ) : null}
          </Box>
        </Box>
      </Stack>
    </PageFrame>
  );
}

function ProductRow(props: {
  product: ApiCatalogProduct;
  first: boolean;
  selected: boolean;
  quantity: number;
  availableForCategory: number | null;
  onChoose: () => void;
}) {
  const { product } = props;
  const packages = Math.ceil(props.quantity / Math.max(product.devices_per_listing, 1));
  const cost = product.price * packages;
  const exceedsBudget = props.availableForCategory !== null && cost > props.availableForCategory;
  const attributes = Object.entries(product.device_attributes)
    .filter(([, value]) => value !== null && value !== undefined && value !== "")
    .slice(0, 3);

  return (
    <Box
      sx={{
        display: "grid",
        gridTemplateColumns: {
          xs: "72px minmax(0, 1fr)",
          md: "88px minmax(0, 1fr) 150px",
        },
        gap: { xs: 1.5, md: 2 },
        alignItems: "center",
        p: { xs: 1.75, sm: 2.25 },
        borderTop: props.first ? "none" : "1px solid rgba(148,163,184,0.24)",
        backgroundColor: props.selected ? "rgba(22,163,74,0.04)" : "#fff",
        color: "#111827",
      }}
    >
      <Box
        sx={{
          width: { xs: 72, md: 88 },
          height: { xs: 72, md: 88 },
          display: "grid",
          placeItems: "center",
          overflow: "hidden",
          borderRadius: 2,
          backgroundColor: "rgba(15,23,42,0.05)",
        }}
      >
        {product.image_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={product.image_url}
            alt=""
            style={{ width: "100%", height: "100%", objectFit: "contain", display: "block" }}
          />
        ) : (
          <DevicesOtherRoundedIcon color="action" />
        )}
      </Box>

      <Box sx={{ minWidth: 0 }}>
        <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap sx={{ mb: 0.5 }}>
          <Typography sx={{ fontWeight: 850, overflowWrap: "anywhere" }}>{product.name}</Typography>
          {props.selected && <Chip size="small" color="success" icon={<CheckRoundedIcon />} label="Выбрано" />}
        </Stack>
        <Typography variant="body2" sx={{ color: "#4b5563" }}>
          {[product.brand, product.model].filter(Boolean).join(" · ")}
        </Typography>

        <Stack direction="row" spacing={0.75} flexWrap="wrap" useFlexGap sx={{ mt: 1 }}>
          {product.rating > 0 && (
            <Chip
              size="small"
              icon={<StarRoundedIcon sx={{ color: "#ca8a04 !important" }} />}
              label={`${product.rating.toFixed(1)} (${product.review_count})`}
              variant="outlined"
            />
          )}
          {product.ecosystems.map((item) => (
            <Chip key={`ecosystem-${item}`} size="small" label={humanizeCategory(item)} variant="outlined" />
          ))}
          {product.protocols.map((item) => (
            <Chip key={`protocol-${item}`} size="small" label={item.toUpperCase()} variant="outlined" />
          ))}
        </Stack>

        {attributes.length > 0 && (
          <Typography
            variant="body2"
            sx={{ mt: 1, color: "#4b5563", overflowWrap: "anywhere" }}
          >
            {attributes.map(([key, value]) => `${humanizeCategory(key)}: ${formatAttribute(value)}`).join(" · ")}
          </Typography>
        )}
      </Box>

      <Box
        sx={{
          gridColumn: { xs: "1 / -1", md: "auto" },
          textAlign: { xs: "left", md: "right" },
        }}
      >
        <Typography variant="h6" sx={{ fontWeight: 900 }}>
          {formatPrice(product.price)} ₽
        </Typography>
        {props.quantity > 1 && (
          <Typography variant="body2" sx={{ color: "#4b5563" }}>
            {formatPrice(cost)} ₽ за {props.quantity} шт.
          </Typography>
        )}
        {product.merchant && (
          <Typography variant="body2" sx={{ color: "#4b5563" }} noWrap title={product.merchant}>
            {product.merchant}
          </Typography>
        )}
        <Button
          variant={props.selected ? "outlined" : "contained"}
          disabled={exceedsBudget}
          onClick={props.onChoose}
          startIcon={props.selected ? <CheckRoundedIcon /> : undefined}
          fullWidth
          sx={{ mt: 1.25, minHeight: 40 }}
        >
          {props.selected ? "Выбрано" : "Добавить"}
        </Button>
        {exceedsBudget && (
          <Typography variant="caption" color="error" sx={{ display: "block", mt: 0.5 }}>
            Превышает бюджет
          </Typography>
        )}
      </Box>
    </Box>
  );
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <Box sx={{ minHeight: "100vh", px: { xs: 2, md: 4 }, py: { xs: 3, md: 5 } }}>
      <Box sx={{ maxWidth: 1180, mx: "auto" }}>{children}</Box>
    </Box>
  );
}

function CenteredProgress() {
  return (
    <Box sx={{ minHeight: "100vh", display: "grid", placeItems: "center" }}>
      <CircularProgress />
    </Box>
  );
}

function optionalPrice(value: string) {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : undefined;
}

function readStoredBudget() {
  if (typeof window === "undefined") return null;
  const value = Number(localStorage.getItem("planner-last-budget"));
  return Number.isFinite(value) && value > 0 ? value : null;
}

function humanizeCategory(value: string) {
  return value
    .split(/[_-]+/)
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(" ");
}

function formatAttribute(value: unknown) {
  if (Array.isArray(value)) return value.join(", ");
  if (typeof value === "boolean") return value ? "да" : "нет";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function formatPrice(value: number) {
  return Math.round(value).toLocaleString("ru-RU");
}
