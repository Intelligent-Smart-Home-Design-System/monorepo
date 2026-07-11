"use client";

/* eslint-disable react-hooks/set-state-in-effect */

import { useEffect, useMemo, useState, type ChangeEvent } from "react";
import { useRouter } from "next/navigation";
import AddRoundedIcon from "@mui/icons-material/AddRounded";
import ExpandMoreRoundedIcon from "@mui/icons-material/ExpandMoreRounded";
import DeleteOutlineRoundedIcon from "@mui/icons-material/DeleteOutlineRounded";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Collapse,
  Divider,
  IconButton,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth-context";
import tracksConfig from "./tracks.json";
import type {
  ApiDeviceType,
  ApiEcosystem,
  ApiFilterOperation,
  ApiRequirementFilter,
  ApiStartPipelineRequest,
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
  planFileType?: "dxf" | "";
  floorJson?: unknown;
  parsedFloor?: unknown;
};

type TrackLevel = {
  name: string;
  description: string;
  price_range: {
    min: number;
    max: number;
  };
  devices: string[];
  max_device_counts?: Record<string, number>;
  device_filters?: Record<string, Record<string, unknown>>;
};

type TrackConfig = {
  name: string;
  levels: Record<string, TrackLevel>;
};

const tracks = (tracksConfig as { tracks: Record<string, TrackConfig> }).tracks;
const trackOptions = Object.entries(tracks).map(([id, track]) => ({ id, ...track }));

export default function SettingsPage() {
  const auth = useAuth();
  const router = useRouter();

  const [budget, setBudget] = useState("500000");
  const [ecosystems, setEcosystems] = useState<ApiEcosystem[]>([]);
  const [deviceTypes, setDeviceTypes] = useState<ApiDeviceType[]>([]);
  const [mainEcosystemId, setMainEcosystemId] = useState("");
  const [selectedLevelByTrack, setSelectedLevelByTrack] = useState<Record<string, string>>({});
  const [expandedTrackId, setExpandedTrackId] = useState("");
  const [requirementsExpandedByTrack, setRequirementsExpandedByTrack] = useState<Record<string, boolean>>({});
  const [requirementsByTrack, setRequirementsByTrack] = useState<Record<string, RequirementDraft[]>>({});
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const [fileName, setFileName] = useState<string>("");
  const [planDataUrl, setPlanDataUrl] = useState<string>("");
  const [planFileType, setPlanFileType] = useState<"dxf" | "">("");
  const [parsedFloor, setParsedFloor] = useState<unknown>(null);
  const [parsingFloor, setParsingFloor] = useState(false);
  const [uploadError, setUploadError] = useState("");

  useEffect(() => {
    if (auth.loading) {
      return;
    }

    if (!auth.isAuthenticated) {
      setLoading(false);
      return;
    }

    let active = true;

    Promise.all([api.listEcosystems(), api.listDeviceTypes()])
      .then(([ecosystemsResponse, deviceTypesResponse]) => {
        if (!active) return;
        const mainEcosystems = ecosystemsResponse.filter((item) => item.may_be_main);
        setEcosystems(mainEcosystems);
        setDeviceTypes(deviceTypesResponse);
        setMainEcosystemId(mainEcosystems[0]?.id ?? "");
        setSelectedLevelByTrack({});
        setExpandedTrackId(trackOptions[0]?.id ?? "");
        setRequirementsByTrack({});
        setRequirementsExpandedByTrack({});
      })
      .catch((err: unknown) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Не удалось загрузить настройки.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [auth.isAuthenticated, auth.loading]);

  const selectedTrackSelections = useMemo(
    () =>
      trackOptions
        .map((track) => {
          const levelId = selectedLevelByTrack[track.id];
          const level = getTrackLevels(track).find((item) => item.id === levelId);
          return level ? { track, levelId, level: level.level } : null;
        })
        .filter((item): item is { track: TrackConfig & { id: string }; levelId: string; level: TrackLevel } =>
          Boolean(item)
        ),
    [selectedLevelByTrack]
  );

  const selectedRequirements = useMemo(
    () => selectedTrackSelections.flatMap(({ track }) => requirementsByTrack[track.id] ?? []),
    [requirementsByTrack, selectedTrackSelections]
  );

  const canSubmit =
    Number(budget) > 0 &&
    mainEcosystemId.length > 0 &&
    !parsingFloor &&
    Boolean(parsedFloor) &&
    selectedRequirements.some((item) => item.device_type && item.quantity > 0);

  const planPreviewState: UploadedPlanState = useMemo(
    () => ({ fileName, planDataUrl, planFileType, floorJson: parsedFloor ?? undefined, parsedFloor: parsedFloor ?? undefined }),
    [fileName, parsedFloor, planDataUrl, planFileType]
  );

  const applyLevel = (trackId: string, levelId: string) => {
    const track = trackOptions.find((item) => item.id === trackId);
    const level = track ? getTrackLevels(track).find((item) => item.id === levelId) : null;
    setSelectedLevelByTrack((prev) => ({ ...prev, [trackId]: levelId }));
    setRequirementsExpandedByTrack((prev) => ({ ...prev, [trackId]: false }));
    setRequirementsByTrack((prev) => ({
      ...prev,
      [trackId]: level ? trackLevelToRequirementDrafts(level.level) : [],
    }));
  };

  const clearTrack = (trackId: string) => {
    setSelectedLevelByTrack((prev) => {
      const next = { ...prev };
      delete next[trackId];
      return next;
    });
    setRequirementsExpandedByTrack((prev) => {
      const next = { ...prev };
      delete next[trackId];
      return next;
    });
    setRequirementsByTrack((prev) => {
      const next = { ...prev };
      delete next[trackId];
      return next;
    });
  };

  const updateTrackRequirements = (
    trackId: string,
    updater: (prev: RequirementDraft[]) => RequirementDraft[]
  ) => {
    setRequirementsByTrack((prev) => ({
      ...prev,
      [trackId]: updater(prev[trackId] ?? []),
    }));
  };

  const handleCreatePlan = async () => {
    setSubmitting(true);
    setError("");

    try {
      if (!parsedFloor || typeof parsedFloor !== "object") {
        setError("Загрузите и распознайте DXF-план перед запуском подбора.");
        return;
      }

      const selectedLevels = Object.fromEntries(
        Object.entries(selectedLevelByTrack).filter(([, levelId]) => levelId)
      );
      const payload: ApiStartPipelineRequest = {
        request_id: crypto.randomUUID(),
        floor_plan: parsedFloor as Record<string, unknown>,
        selected_levels: selectedLevels,
        device_selection: {
          main_ecosystem: mainEcosystemId,
          budget: Number(budget),
          max_solutions: 5,
          time_budget_seconds: 10,
          requirements: selectedRequirements
            .filter((item) => item.device_type && item.quantity > 0)
            .map((item, index) => ({
              requirement_id: index + 1,
              device_type: item.device_type,
              count: item.quantity,
              connect_to_main_ecosystem: true,
              filters: item.filters,
            })),
        },
      };

      const started = await api.startPipeline(payload);

      localStorage.setItem(
        "planner-uploaded-plan",
        JSON.stringify(planPreviewState)
      );
      localStorage.setItem("planner-last-budget", budget);

      const params = new URLSearchParams({ workflow_id: started.workflow_id });
      if (started.run_id) params.set("run_id", started.run_id);
      router.push(`/plan?${params.toString()}`);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Не удалось создать новый план.");
    } finally {
      setSubmitting(false);
    }
  };

  const handleFloorFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    setFileName(file.name);
    setUploadError("");
    setParsedFloor(null);

    const lowerName = file.name.toLowerCase();
    if (!lowerName.endsWith(".dxf")) {
      setPlanDataUrl("");
      setPlanFileType("");
      setUploadError("Сейчас поддерживаются только файлы DXF.");
      return;
    }

    setPlanDataUrl("");
    setPlanFileType("dxf");
    setParsingFloor(true);

    try {
      const floor = await api.parseFloorPlan(file);
      setParsedFloor(floor);
    } catch (err: unknown) {
      setUploadError(err instanceof Error ? err.message : "Не удалось распознать DXF-файл.");
    } finally {
      setParsingFloor(false);
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
                Выберите бюджет, основную экосистему и уровень подбора для будущего плана.
              </Typography>
            </Box>

            {loading ? (
              <Box sx={{ py: 8, display: "grid", placeItems: "center" }}>
                <CircularProgress />
              </Box>
            ) : !auth.isAuthenticated ? (
              <Alert
                severity="warning"
                action={
                  <Button color="inherit" size="small" onClick={() => router.push("/login?next=/settings")}>
                    Войти
                  </Button>
                }
              >
                Для создания плана нужно войти в аккаунт.
              </Alert>
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
                  <Typography sx={{ fontWeight: 700, mb: 1 }}>Треки подбора</Typography>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1.4 }}>
                    Выберите уровень отдельно для каждого направления, которое должно попасть в план.
                  </Typography>

                  <Stack spacing={2}>
                    {trackOptions.map((track) => {
                      const selectedLevelId = selectedLevelByTrack[track.id] ?? "";
                      const trackLevels = getTrackLevels(track);
                      const selectedLevel = trackLevels.find((item) => item.id === selectedLevelId);
                      const trackRequirements = requirementsByTrack[track.id] ?? [];
                      const trackExpanded = expandedTrackId === track.id;
                      const requirementsExpanded = Boolean(requirementsExpandedByTrack[track.id]);
                      const accent = getTrackAccent(track.id);

                      return (
                        <Box
                          key={track.id}
                          sx={{
                            borderRadius: 4,
                            border: selectedLevel
                              ? `2px solid ${accent.main}`
                              : "1px solid rgba(148,163,184,0.32)",
                            background: selectedLevel
                              ? `linear-gradient(135deg, ${accent.soft}, #fff)`
                              : "#fff",
                            overflow: "hidden",
                          }}
                        >
                          <Stack
                            direction={{ xs: "column", sm: "row" }}
                            justifyContent="space-between"
                            alignItems={{ xs: "stretch", sm: "center" }}
                            spacing={1}
                            onClick={() => setExpandedTrackId((current) => (current === track.id ? "" : track.id))}
                            sx={{ p: 2, cursor: "pointer" }}
                          >
                            <Stack direction="row" spacing={1.4} alignItems="center">
                              <Box
                                sx={{
                                  width: 10,
                                  alignSelf: "stretch",
                                  minHeight: 52,
                                  borderRadius: 999,
                                  backgroundColor: accent.main,
                                }}
                              />
                              <Box>
                                <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
                                <Typography sx={{ fontWeight: 850 }}>{track.name}</Typography>
                                  {selectedLevel && (
                                    <Chip
                                      size="small"
                                      label="В плане"
                                      sx={{ color: accent.main, backgroundColor: accent.soft, fontWeight: 700 }}
                                    />
                                  )}
                                </Stack>
                                <Typography variant="body2" color="text.secondary">
                                  {selectedLevel
                                    ? `Выбран уровень: ${selectedLevel.level.name}`
                                    : "Нажмите, чтобы выбрать уровень для этого трека."}
                                </Typography>
                              </Box>
                            </Stack>

                            <Stack direction="row" spacing={1} alignItems="center" onClick={(event) => event.stopPropagation()}>
                              {selectedLevel && (
                                <Button variant="outlined" size="small" onClick={() => clearTrack(track.id)}>
                                  Не использовать
                                </Button>
                              )}
                              <IconButton
                                onClick={() => setExpandedTrackId((current) => (current === track.id ? "" : track.id))}
                                sx={{
                                  color: accent.main,
                                  transform: trackExpanded ? "rotate(180deg)" : "rotate(0deg)",
                                  transition: "transform 180ms ease",
                                }}
                              >
                                <ExpandMoreRoundedIcon />
                              </IconButton>
                            </Stack>
                          </Stack>

                          <Collapse in={trackExpanded} timeout="auto" unmountOnExit>
                            <Box sx={{ px: 2, pb: 2 }}>
                              <Stack spacing={1.2}>
                                {trackLevels.map((item) => {
                                  const active = item.id === selectedLevelId;
                                  return (
                                    <Box
                                      key={item.id}
                                      onClick={() => applyLevel(track.id, item.id)}
                                      sx={{
                                        cursor: "pointer",
                                        borderRadius: 3,
                                        p: 1.5,
                                        border: active
                                          ? `2px solid ${accent.main}`
                                          : "1px solid rgba(148,163,184,0.32)",
                                        background: active ? accent.soft : "rgba(255,255,255,0.86)",
                                      }}
                                    >
                                      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
                                        <Typography sx={{ fontWeight: 800 }}>{item.level.name}</Typography>
                                        {active && (
                                          <Chip
                                            size="small"
                                            label="Выбран"
                                            sx={{ color: accent.main, backgroundColor: "#fff", fontWeight: 700 }}
                                          />
                                        )}
                                      </Stack>

                                      <Typography variant="body2" color="text.secondary">
                                        {item.level.description}
                                      </Typography>

                                      <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap", mt: 1 }}>
                                        <Chip
                                          size="small"
                                          label={`${formatPrice(item.level.price_range.min)}-${formatPrice(item.level.price_range.max)} ₽`}
                                        />
                                        <Chip size="small" label={`Типов устройств: ${item.level.devices.length}`} />
                                      </Stack>
                                    </Box>
                                  );
                                })}
                              </Stack>

                              {selectedLevel ? (
                                <Box sx={{ mt: 2 }}>
                                  <Stack
                                    direction={{ xs: "column", sm: "row" }}
                                    justifyContent="space-between"
                                    alignItems={{ xs: "stretch", sm: "center" }}
                                    spacing={1}
                                    sx={{ mb: 1.2 }}
                                  >
                                    <Box>
                                      <Typography sx={{ fontWeight: 700 }}>
                                        Требования трека «{track.name}»
                                      </Typography>
                                      <Typography variant="body2" color="text.secondary">
                                        Эти устройства добавятся в общий список требований плана.
                                      </Typography>
                                    </Box>

                                    <Button
                                      variant="outlined"
                                      onClick={() =>
                                        setRequirementsExpandedByTrack((prev) => ({
                                          ...prev,
                                          [track.id]: !prev[track.id],
                                        }))
                                      }
                                      sx={{ borderRadius: 3 }}
                                    >
                                      {requirementsExpanded ? "Скрыть требования" : "Раскрыть требования"}
                                    </Button>
                                  </Stack>

                                  <Collapse in={requirementsExpanded} timeout="auto" unmountOnExit>
                                    <Stack spacing={1.6}>
                                      {trackRequirements.map((requirement, index) => {
                                        const selectedType = deviceTypes.find(
                                          (deviceType) => deviceType.id === requirement.device_type
                                        );

                                        return (
                                          <Card key={requirement.localId} variant="outlined" sx={{ borderRadius: 4 }}>
                                            <CardContent>
                                              <Stack spacing={1.4}>
                                                <Stack direction="row" justifyContent="space-between" alignItems="center">
                                                  <Typography sx={{ fontWeight: 800 }}>
                                                    {track.name}: требование #{index + 1}
                                                  </Typography>
                                                  <IconButton
                                                    onClick={() =>
                                                      updateTrackRequirements(track.id, (prev) =>
                                                        prev.filter((item) => item.localId !== requirement.localId)
                                                      )
                                                    }
                                                    disabled={trackRequirements.length === 1}
                                                  >
                                                    <DeleteOutlineRoundedIcon />
                                                  </IconButton>
                                                </Stack>

                                                <Select
                                                  fullWidth
                                                  value={requirement.device_type}
                                                  onChange={(event) =>
                                                    updateTrackRequirements(track.id, (prev) =>
                                                      prev.map((item) =>
                                                        item.localId === requirement.localId
                                                          ? { ...item, device_type: String(event.target.value), filters: [] }
                                                          : item
                                                      )
                                                    )
                                                  }
                                                >
                                                  {!selectedType && requirement.device_type && (
                                                    <MenuItem value={requirement.device_type}>
                                                      {requirement.device_type}
                                                    </MenuItem>
                                                  )}
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
                                                    updateTrackRequirements(track.id, (prev) =>
                                                      prev.map((item) =>
                                                        item.localId === requirement.localId
                                                          ? {
                                                              ...item,
                                                              quantity: Math.max(1, Number(event.target.value) || 1),
                                                            }
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
                                                          updateTrackRequirements(track.id, (prev) =>
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
                                                    Для этого типа устройства нет дополнительных фильтров.
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
                                        updateTrackRequirements(track.id, (prev) => [
                                          ...prev,
                                          makeEmptyRequirement(deviceTypes[0]?.id ?? ""),
                                        ])
                                      }
                                    >
                                      Добавить требование в трек «{track.name}»
                                    </Button>
                                  </Collapse>
                                </Box>
                              ) : null}
                            </Box>
                          </Collapse>
                        </Box>
                      );
                    })}
                  </Stack>

                  {selectedTrackSelections.length === 0 && (
                    <Alert severity="info" sx={{ mt: 1.5 }}>
                      Выберите уровень хотя бы в одном треке, чтобы запустить подбор.
                    </Alert>
                  )}
                </Box>

                <Box>
                  <Typography sx={{ fontWeight: 700, mb: 1 }}>План квартиры</Typography>
                  <Button variant="outlined" component="label">
                    Загрузить файл DXF
                    <input
                      hidden
                      type="file"
                      accept=".dxf"
                      onChange={handleFloorFileChange}
                    />
                  </Button>

                  <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                    Добавьте план квартиры, чтобы использовать его на следующих шагах подбора.
                  </Typography>

                  {fileName && (
                    <Alert sx={{ mt: 1.2 }} severity={uploadError ? "error" : parsingFloor ? "info" : "success"}>
                      {uploadError ||
                        (parsingFloor
                          ? `Распознаём план: ${fileName}`
                          : parsedFloor
                            ? `План распознан: ${fileName}`
                            : `Загружено: ${fileName}`)}
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
                {selectedTrackSelections.length === 0 && (
                  <Typography variant="body2" color="text.secondary" textAlign="center">
                    Кнопка активируется после выбора уровня и распознавания DXF-плана.
                  </Typography>
                )}
                {selectedTrackSelections.length > 0 && !parsedFloor && (
                  <Typography variant="body2" color="text.secondary" textAlign="center">
                    Загрузите DXF-файл, чтобы передать план квартиры в pipeline.
                  </Typography>
                )}
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

function getTrackLevels(track: TrackConfig & { id: string }) {
  return Object.entries(track.levels)
    .sort(([left], [right]) => Number(left) - Number(right))
    .map(([id, level]) => ({ id, level }));
}

function trackLevelToRequirementDrafts(level: TrackLevel): RequirementDraft[] {
  return level.devices.map((deviceType, index) => ({
    localId: crypto.randomUUID(),
    device_type: deviceType,
    quantity: level.max_device_counts?.[deviceType] ?? 1,
    filters: Object.entries(level.device_filters?.[deviceType] ?? {}).map(([field, value]) => ({
      field,
      operation: Array.isArray(value) ? "contains" : "eq",
      value: Array.isArray(value) ? value[0] : normalizeTrackFilterValue(value),
    })),
  }));
}

function normalizeTrackFilterValue(value: unknown) {
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return value;
  }
  return String(value);
}

function formatPrice(value: number) {
  return Math.round(value).toLocaleString("ru-RU");
}

function getTrackAccent(trackId: string) {
  const accents: Record<string, { main: string; soft: string }> = {
    security: { main: "#dc2626", soft: "rgba(220,38,38,0.09)" },
    light: { main: "#ca8a04", soft: "rgba(250,204,21,0.18)" },
    climate: { main: "#0891b2", soft: "rgba(8,145,178,0.10)" },
    appliances: { main: "#7c3aed", soft: "rgba(124,58,237,0.09)" },
    entertainment: { main: "#0f766e", soft: "rgba(15,118,110,0.10)" },
  };

  return accents[trackId] ?? { main: "#2563eb", soft: "rgba(37,99,235,0.10)" };
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
