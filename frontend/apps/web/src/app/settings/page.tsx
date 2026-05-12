"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import AddRoundedIcon from "@mui/icons-material/AddRounded";
import DeleteOutlineRoundedIcon from "@mui/icons-material/DeleteOutlineRounded";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  IconButton,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import { api } from "../lib/api";
import type {
  ApiCreatePlanRequest,
  ApiDeviceType,
  ApiEcosystem,
  ApiFilterOperation,
  ApiPreset,
  ApiRequirementFilter,
} from "../lib/types";

type RequirementDraft = {
  localId: string;
  device_type: string;
  quantity: number;
  filters: ApiRequirementFilter[];
};

type UploadedPlanState = {
  fileName?: string;
  planDataUrl?: string;
  planFileType?: "dxf" | "png" | "";
};

export default function SettingsPage() {
  const router = useRouter();

  const [budget, setBudget] = useState("500000");
  const [ecosystems, setEcosystems] = useState<ApiEcosystem[]>([]);
  const [presets, setPresets] = useState<ApiPreset[]>([]);
  const [deviceTypes, setDeviceTypes] = useState<ApiDeviceType[]>([]);
  const [mainEcosystemId, setMainEcosystemId] = useState("");
  const [requirements, setRequirements] = useState<RequirementDraft[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const [fileName, setFileName] = useState<string>("");
  const [planDataUrl, setPlanDataUrl] = useState<string>("");
  const [planFileType, setPlanFileType] = useState<"dxf" | "png" | "">("");
  const [uploadError, setUploadError] = useState("");

  useEffect(() => {
    let active = true;

    Promise.all([api.listEcosystems(), api.listPresets(), api.listDeviceTypes()])
      .then(([ecosystemsResponse, presetsResponse, deviceTypesResponse]) => {
        if (!active) return;
        const mainEcosystems = ecosystemsResponse.filter((item) => item.may_be_main);
        setEcosystems(mainEcosystems);
        setPresets(presetsResponse);
        setDeviceTypes(deviceTypesResponse);
        setMainEcosystemId(mainEcosystems[0]?.id ?? "");
        setRequirements([makeEmptyRequirement(deviceTypesResponse[0]?.id ?? "")]);
      })
      .catch((err: unknown) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Не удалось загрузить настройки из backend.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, []);

  const canSubmit =
    Number(budget) > 0 &&
    mainEcosystemId.length > 0 &&
    requirements.some((item) => item.device_type && item.quantity > 0);

  const planPreviewState: UploadedPlanState = useMemo(
    () => ({ fileName, planDataUrl, planFileType }),
    [fileName, planDataUrl, planFileType]
  );

  const applyPreset = (preset: ApiPreset) => {
    setRequirements(
      preset.requirements.map((requirement) => ({
        localId: crypto.randomUUID(),
        device_type: requirement.device_type,
        quantity: requirement.quantity,
        filters: requirement.filters ?? [],
      }))
    );
  };

  const handleCreatePlan = async () => {
    setSubmitting(true);
    setError("");

    try {
      const payload: ApiCreatePlanRequest = {
        budget: Number(budget),
        main_ecosystem_id: mainEcosystemId,
        requirements: requirements
          .filter((item) => item.device_type && item.quantity > 0)
          .map((item) => ({
            device_type: item.device_type,
            quantity: item.quantity,
            filters: item.filters,
          })),
      };

      const created = await api.createPlan(payload);

      localStorage.setItem(
        "planner-uploaded-plan",
        JSON.stringify(planPreviewState)
      );

      router.push(`/plan?id=${created.plan_id}`);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Не удалось создать новый план.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Box sx={{ minHeight: "100vh", px: { xs: 2, md: 4 }, py: { xs: 3, md: 5 } }}>
      <Card sx={{ maxWidth: 900, mx: "auto", borderRadius: 6, overflow: "hidden" }}>
        <CardContent>
          <Stack spacing={3}>
            <Box>
              <Typography variant="h4" sx={{ fontWeight: 800, mb: 1 }}>
                Ввод настроек
              </Typography>
              <Typography color="text.secondary">
                Эта страница уже работает с backend API: экосистемы, пресеты и типы устройств
                загружаются по сети, а запуск создаёт реальный план через `POST /api/v1/plans`.
              </Typography>
            </Box>

            {loading ? (
              <Box sx={{ py: 8, display: "grid", placeItems: "center" }}>
                <CircularProgress />
              </Box>
            ) : (
              <>
                {error && <Alert severity="error">{error}</Alert>}

                <TextField
                  label="Бюджет (₽)"
                  value={budget}
                  onChange={(event) => setBudget(event.target.value.replace(/[^\d]/g, ""))}
                  inputMode="numeric"
                  fullWidth
                />

                <Box>
                  <Typography sx={{ fontWeight: 700, mb: 1.2 }}>Выбор основной экосистемы</Typography>
                  <Stack spacing={1.2}>
                    {ecosystems.map((ecosystem) => {
                      const active = ecosystem.id === mainEcosystemId;
                      return (
                        <Box
                          key={ecosystem.id}
                          onClick={() => setMainEcosystemId(ecosystem.id)}
                          sx={{
                            cursor: "pointer",
                            borderRadius: 4,
                            p: 2,
                            border: active
                              ? "2px solid #2563eb"
                              : "1px solid rgba(148,163,184,0.28)",
                            background: active
                              ? "linear-gradient(135deg, rgba(37,99,235,0.10), rgba(59,130,246,0.04))"
                              : "#fff",
                          }}
                        >
                          <Stack direction="row" spacing={2} alignItems="center">
                            <Box
                              sx={{
                                width: 52,
                                height: 52,
                                borderRadius: 3,
                                overflow: "hidden",
                                display: "grid",
                                placeItems: "center",
                                backgroundColor: active ? "rgba(37,99,235,0.10)" : "rgba(15,23,42,0.05)",
                              }}
                            >
                              {ecosystem.image_url ? (
                                // eslint-disable-next-line @next/next/no-img-element
                                <img
                                  src={ecosystem.image_url}
                                  alt={ecosystem.name}
                                  style={{ width: 34, height: 34, objectFit: "contain", display: "block" }}
                                />
                              ) : (
                                <Typography sx={{ fontWeight: 800, color: "#1d4ed8" }}>
                                  {ecosystem.name.slice(0, 1)}
                                </Typography>
                              )}
                            </Box>

                            <Box sx={{ flex: 1 }}>
                              <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.4 }}>
                                <Typography sx={{ fontWeight: 800 }}>{ecosystem.name}</Typography>
                                {active && <Chip size="small" label="Выбрано" color="primary" />}
                              </Stack>
                              <Typography variant="body2" color="text.secondary">
                                {ecosystem.description}
                              </Typography>
                            </Box>
                          </Stack>
                        </Box>
                      );
                    })}
                  </Stack>
                </Box>

                <Box>
                  <Typography sx={{ fontWeight: 700, mb: 1 }}>Presets</Typography>
                  <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap" }}>
                    {presets.map((preset) => (
                      <Button
                        key={preset.id}
                        variant="outlined"
                        onClick={() => applyPreset(preset)}
                        sx={{ borderRadius: 3 }}
                      >
                        {preset.name}
                      </Button>
                    ))}
                  </Stack>
                </Box>

                <Box>
                  <Typography sx={{ fontWeight: 700, mb: 1.2 }}>Требования</Typography>
                  <Stack spacing={1.6}>
                    {requirements.map((requirement, index) => {
                      const selectedType = deviceTypes.find(
                        (deviceType) => deviceType.id === requirement.device_type
                      );

                      return (
                        <Card key={requirement.localId} variant="outlined" sx={{ borderRadius: 4 }}>
                          <CardContent>
                            <Stack spacing={1.4}>
                              <Stack direction="row" justifyContent="space-between" alignItems="center">
                                <Typography sx={{ fontWeight: 800 }}>
                                  Требование #{index + 1}
                                </Typography>
                                <IconButton
                                  onClick={() =>
                                    setRequirements((prev) => prev.filter((item) => item.localId !== requirement.localId))
                                  }
                                  disabled={requirements.length === 1}
                                >
                                  <DeleteOutlineRoundedIcon />
                                </IconButton>
                              </Stack>

                              <Select
                                fullWidth
                                value={requirement.device_type}
                                onChange={(event) =>
                                  setRequirements((prev) =>
                                    prev.map((item) =>
                                      item.localId === requirement.localId
                                        ? { ...item, device_type: String(event.target.value), filters: [] }
                                        : item
                                    )
                                  )
                                }
                              >
                                {deviceTypes.map((deviceType) => (
                                  <MenuItem key={deviceType.id} value={deviceType.id}>
                                    {deviceType.name}
                                  </MenuItem>
                                ))}
                              </Select>

                              <TextField
                                label="Количество"
                                type="number"
                                value={requirement.quantity}
                                onChange={(event) =>
                                  setRequirements((prev) =>
                                    prev.map((item) =>
                                      item.localId === requirement.localId
                                        ? { ...item, quantity: Math.max(1, Number(event.target.value) || 1) }
                                        : item
                                    )
                                  )
                                }
                                inputProps={{ min: 1 }}
                              />

                              {selectedType?.filters?.length ? (
                                <Stack spacing={1}>
                                  <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                    Доступные фильтры
                                  </Typography>
                                  {selectedType.filters.map((filterField) => (
                                    <FilterEditor
                                      key={`${requirement.localId}-${filterField.field}`}
                                      filterField={filterField}
                                      value={requirement.filters.find((item) => item.field === filterField.field)}
                                      onChange={(nextFilter) =>
                                        setRequirements((prev) =>
                                          prev.map((item) =>
                                            item.localId === requirement.localId
                                              ? {
                                                  ...item,
                                                  filters: mergeFilters(item.filters, nextFilter),
                                                }
                                              : item
                                          )
                                        )
                                      }
                                    />
                                  ))}
                                </Stack>
                              ) : (
                                <Typography variant="body2" color="text.secondary">
                                  Для этого типа устройства backend не прислал доступных фильтров.
                                </Typography>
                              )}
                            </Stack>
                          </CardContent>
                        </Card>
                      );
                    })}
                  </Stack>

                  <Button
                    sx={{ mt: 1.5 }}
                    startIcon={<AddRoundedIcon />}
                    variant="outlined"
                    onClick={() =>
                      setRequirements((prev) => [...prev, makeEmptyRequirement(deviceTypes[0]?.id ?? "")])
                    }
                  >
                    Добавить требование
                  </Button>
                </Box>

                <Box>
                  <Typography sx={{ fontWeight: 700, mb: 1 }}>План квартиры</Typography>
                  <Button variant="outlined" component="label">
                    Загрузить файл DXF/PNG
                    <input
                      hidden
                      type="file"
                      accept=".dxf,.png,image/png"
                      onChange={(event) => {
                        const file = event.target.files?.[0];
                        if (!file) return;

                        setFileName(file.name);
                        setUploadError("");

                        const lowerName = file.name.toLowerCase();
                        const isPng = lowerName.endsWith(".png") || file.type === "image/png";
                        const isDxf = lowerName.endsWith(".dxf");

                        if (!isPng && !isDxf) {
                          setUploadError("Сейчас поддерживаем только файлы DXF или PNG.");
                          return;
                        }

                        if (isPng) {
                          const reader = new FileReader();
                          reader.onload = () => {
                            setPlanDataUrl(String(reader.result));
                            setPlanFileType("png");
                          };
                          reader.readAsDataURL(file);
                        } else {
                          setPlanDataUrl("");
                          setPlanFileType("dxf");
                        }
                      }}
                    />
                  </Button>

                  <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                    В текущем API файл плана ещё не отправляется на backend, но сохраняется локально
                    для предпросмотра на странице результата.
                  </Typography>

                  {fileName && (
                    <Alert sx={{ mt: 1.2 }} severity={uploadError ? "error" : "success"}>
                      {uploadError || `Загружено: ${fileName}`}
                    </Alert>
                  )}
                </Box>

                <Divider />

                <Button
                  variant="contained"
                  fullWidth
                  size="large"
                  disabled={!canSubmit || submitting}
                  onClick={handleCreatePlan}
                >
                  {submitting ? "Создаём план..." : "Запустить подбор"}
                </Button>
              </>
            )}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  );
}

function makeEmptyRequirement(deviceTypeId: string): RequirementDraft {
  return {
    localId: crypto.randomUUID(),
    device_type: deviceTypeId,
    quantity: 1,
    filters: [],
  };
}

function mergeFilters(
  filters: ApiRequirementFilter[],
  nextFilter: ApiRequirementFilter | null
): ApiRequirementFilter[] {
  if (!nextFilter) return filters;
  const withoutCurrent = filters.filter((item) => item.field !== nextFilter.field);
  if (nextFilter.value === "" || nextFilter.value === null || nextFilter.value === undefined) {
    return withoutCurrent;
  }
  return [...withoutCurrent, nextFilter];
}

function FilterEditor(props: {
  filterField: ApiDeviceType["filters"][number];
  value?: ApiRequirementFilter;
  onChange: (nextFilter: ApiRequirementFilter | null) => void;
}) {
  const operation = props.value?.operation ?? props.filterField.operations[0] ?? "eq";
  const currentValue =
    props.value?.value === undefined || props.value?.value === null ? "" : String(props.value.value);

  return (
    <Stack direction={{ xs: "column", md: "row" }} spacing={1}>
      <TextField
        label={props.filterField.name}
        value={currentValue}
        onChange={(event) =>
          props.onChange({
            field: props.filterField.field,
            operation,
            value: castFilterValue(props.filterField.value_type, event.target.value),
          })
        }
        select={Boolean(props.filterField.enum_values?.length)}
        sx={{ flex: 1 }}
      >
        {props.filterField.enum_values?.map((value) => (
          <MenuItem key={value} value={value}>
            {value}
          </MenuItem>
        ))}
      </TextField>

      <Select
        value={operation}
        onChange={(event) =>
          props.onChange({
            field: props.filterField.field,
            operation: event.target.value as ApiFilterOperation,
            value: castFilterValue(props.filterField.value_type, currentValue),
          })
        }
        sx={{ minWidth: 120 }}
      >
        {props.filterField.operations.map((item) => (
          <MenuItem key={item} value={item}>
            {item}
          </MenuItem>
        ))}
      </Select>
    </Stack>
  );
}

function castFilterValue(type: ApiDeviceType["filters"][number]["value_type"], value: string) {
  if (value === "") return "";
  switch (type) {
    case "boolean":
      return value === "true";
    case "integer":
      return Number.parseInt(value, 10);
    case "number":
      return Number(value);
    default:
      return value;
  }
}
