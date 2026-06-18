"use client";

/* eslint-disable react-hooks/set-state-in-effect */

import { Suspense, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Avatar,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  LinearProgress,
  Stack,
  Tab,
  Tabs,
  Typography,
} from "@mui/material";
import Image from "next/image";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth-context";
import type { ApiHomePlan, ApiPipelineResult, ApiPlanStageArtifact, ApiPlanStatus } from "../lib/types";
import { ApartmentPlanPreview } from "./ApartmentPlanPreview";

type UploadedPlanState = {
  fileName?: string;
  planDataUrl?: string;
  planFileType?: "dxf" | "png" | "";
  floorJson?: unknown;
  parsedFloor?: unknown;
  floor?: unknown;
  zones?: unknown;
};

type ResultTabKey = string;

type ResultTab = {
  key: ResultTabKey;
  label: string;
  artifact?: ApiPlanStageArtifact;
};

type PipelineStatusResponse = {
  workflow_id: string;
  run_id?: string;
  status: string;
  stages?: ApiPlanStageArtifact[] | null;
  artifacts?: ApiPlanStageArtifact[] | null;
  parsed_floor_plan?: unknown;
  layout?: unknown;
  device_selection?: unknown;
};

type SimulationDevice = {
  id: string;
  trigger_id: string;
  name: string;
  type: string;
  device_type: string;
  room_id: string;
  position?: {
    x: number;
    y: number;
  };
  direction?: {
    x: number;
    y: number;
  };
  track?: string;
  filters?: Record<string, unknown>;
  listing_id?: number;
  requirement_id?: number;
};

export default function PlanPage() {
  return (
    <Suspense
      fallback={
        <Box sx={{ minHeight: "100vh", display: "grid", placeItems: "center" }}>
          <CircularProgress />
        </Box>
      }
    >
      <PlanPageContent />
    </Suspense>
  );
}

function PlanPageContent() {
  const auth = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();

  const [status, setStatus] = useState<ApiPlanStatus | null>(null);
  const [plan, setPlan] = useState<ApiHomePlan | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [uploadedPlan] = useState<UploadedPlanState | null>(() => loadUploadedPlan());
  const [selectedBundleId, setSelectedBundleId] = useState<number | null>(null);
  const [selectedListingId, setSelectedListingId] = useState<number | null>(null);
  const [activeResultTab, setActiveResultTab] = useState<ResultTabKey>("final");

  const planId = Number(searchParams.get("id") ?? "");
  const workflowId = searchParams.get("workflow_id") ?? "";
  const runId = searchParams.get("run_id") ?? "";
  const hasWorkflowTarget = workflowId.trim().length > 0;
  const hasLegacyPlanTarget = Number.isFinite(planId) && planId > 0;
  const invalidPlanId = !hasWorkflowTarget && !hasLegacyPlanTarget;
  const displayPlanId = hasLegacyPlanTarget ? String(planId) : workflowId || "—";

  useEffect(() => {
    if (auth.loading || !auth.isAuthenticated) {
      if (!auth.loading) {
        setLoading(false);
      }
      return;
    }

    if (invalidPlanId) {
      return;
    }

    let active = true;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const loadStatus = async () => {
      try {
        if (hasWorkflowTarget) {
          const currentResult = await api.getPipelineResult(workflowId, runId || undefined);
          if (!active) return;

          if (isPipelinePending(currentResult)) {
            const normalizedStatus = normalizePipelineStatus(currentResult.status);
            const stages = pipelineResultToStageArtifacts(currentResult);
            setStatus({
              plan_id: 0,
              status: normalizedStatus,
              progress: null,
              stages: stages.length
                ? stages
                : [
                    {
                      key: "workflow",
                      title: "Pipeline",
                      status: currentResult.status,
                      payload: currentResult,
                    },
                  ],
            });
            setPlan(pipelineResultToPlan(currentResult, Number(budgetFromStorage()) || 0));
            setLoading(false);
            if (normalizedStatus === "failed") {
              setError("Pipeline завершился с ошибкой. Подробности статуса показаны в промежуточных этапах.");
              return;
            }
            timer = setTimeout(loadStatus, 4000);
            return;
          }

          const fullPlan = pipelineResultToPlan(currentResult, Number(budgetFromStorage()) || 0);
          setStatus({
            plan_id: 0,
            status: "completed",
            progress: 1,
            stages: pipelineResultToStageArtifacts(currentResult),
          });
          setPlan(fullPlan);
          setSelectedBundleId(fullPlan.bundles[0]?.id ?? null);
          setSelectedListingId(fullPlan.bundles[0]?.listings[0]?.id ?? null);
          setLoading(false);
          return;
        }

        const currentStatus = await api.getPlanStatus(planId);
        if (!active) return;
        setStatus(currentStatus);

        if (currentStatus.status === "completed") {
          const fullPlan = await api.getPlan(planId);
          if (!active) return;
          setPlan(fullPlan);
          setSelectedBundleId(fullPlan.bundles[0]?.id ?? null);
          setSelectedListingId(fullPlan.bundles[0]?.listings[0]?.id ?? null);
          setLoading(false);
          return;
        }

        if (currentStatus.status === "failed") {
          setError(currentStatus.error?.message ?? "Генерация плана завершилась ошибкой.");
          setLoading(false);
          return;
        }

        setLoading(false);
        timer = setTimeout(loadStatus, 4000);
      } catch (err: unknown) {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Не удалось получить статус плана.");
        setLoading(false);
      }
    };

    loadStatus();

    return () => {
      active = false;
      if (timer) clearTimeout(timer);
    };
  }, [auth.isAuthenticated, auth.loading, hasWorkflowTarget, invalidPlanId, planId, runId, workflowId]);

  const selectedBundle = useMemo(
    () => plan?.bundles.find((bundle) => bundle.id === selectedBundleId) ?? plan?.bundles[0] ?? null,
    [plan, selectedBundleId]
  );

  const selectedListing = useMemo(
    () =>
      selectedBundle?.listings.find((listing) => listing.id === selectedListingId) ??
      selectedBundle?.listings[0] ??
      null,
    [selectedBundle, selectedListingId]
  );

  const simulationFloorData = useMemo(
    () => collectSimulationFloorData(uploadedPlan, status, plan),
    [plan, status, uploadedPlan]
  );

  const selectedSimulationDevices = useMemo(
    () => (selectedBundle ? simulationDevicesFromBundle(selectedBundle, simulationFloorData) : []),
    [selectedBundle, simulationFloorData]
  );

  const bundleTotalListings = selectedBundle?.listings.length ?? 0;

  const stageArtifacts = useMemo(
    () => collectStageArtifacts(status, plan),
    [plan, status]
  );

  const resultTabs = useMemo(
    () => buildResultTabs(stageArtifacts, Boolean(plan)),
    [plan, stageArtifacts]
  );

  useEffect(() => {
    if (!resultTabs.length) return;
    if (!resultTabs.some((tab) => tab.key === activeResultTab)) {
      setActiveResultTab(resultTabs[0].key);
    }
  }, [activeResultTab, resultTabs]);

  const activeStageArtifact = resultTabs.find((tab) => tab.key === activeResultTab)?.artifact;

  return (
    <Box
      sx={{
        minHeight: "100vh",
        px: { xs: 2, md: 4 },
        py: { xs: 3, md: 5 },
        background:
          "radial-gradient(circle at 18% 8%, rgba(56,189,248,0.18), transparent 28%), radial-gradient(circle at 86% 18%, rgba(34,197,94,0.16), transparent 30%), linear-gradient(135deg, #020617 0%, #071426 45%, #0f172a 100%)",
      }}
    >
      <Box sx={{ maxWidth: 1180, mx: "auto" }}>
        <Box
          sx={{
            mb: 2.5,
            p: { xs: 2.2, md: 3 },
            borderRadius: 6,
            color: "#fff",
            background:
              "linear-gradient(135deg, rgba(255,255,255,0.16), rgba(255,255,255,0.06))",
            border: "1px solid rgba(255,255,255,0.18)",
            boxShadow: "0 24px 70px rgba(0,0,0,0.22)",
            backdropFilter: "blur(18px)",
          }}
        >
          <Stack
            direction={{ xs: "column", lg: "row" }}
            justifyContent="space-between"
            alignItems={{ xs: "flex-start", lg: "center" }}
            spacing={2.5}
          >
            <Box>
              <Typography variant="h4" sx={{ fontWeight: 900, letterSpacing: 0, mb: 0.8 }}>
                План умного дома #{displayPlanId}
              </Typography>
              <Typography sx={{ color: "rgba(255,255,255,0.72)", maxWidth: 620 }}>
                Следите за готовностью плана и просматривайте подобранные наборы устройств.
              </Typography>
            </Box>

            <Stack direction={{ xs: "column", sm: "row" }} spacing={1.2} sx={{ width: { xs: "100%", sm: "auto" } }}>
              <Button
                variant="contained"
                onClick={() => {
                  if (selectedBundle) {
                    openSimulation(selectedBundle, simulationFloorData);
                  } else {
                    openSimulationFromPlan(hasLegacyPlanTarget ? planId : workflowId, simulationFloorData);
                  }
                }}
                sx={{
                  borderRadius: 3,
                  fontWeight: 900,
                  background: "linear-gradient(135deg, #7c3aed, #c026d3)",
                  boxShadow: "0 14px 34px rgba(124,58,237,0.28)",
                  "&:hover": {
                    background: "linear-gradient(135deg, #6d28d9, #a21caf)",
                    boxShadow: "0 18px 42px rgba(124,58,237,0.34)",
                  },
                }}
              >
                Перейти к симуляции
              </Button>

              <Button
                variant="contained"
                onClick={() => router.push("/")}
                sx={{
                  borderRadius: 3,
                  fontWeight: 900,
                  background: "linear-gradient(135deg, #2563eb, #38bdf8)",
                }}
              >
                На главную
              </Button>
            </Stack>
          </Stack>
        </Box>

        {!auth.isAuthenticated ? (
          <Alert
            severity="warning"
            action={
              <Button color="inherit" size="small" onClick={() => router.push(`/login?next=${encodeURIComponent(window.location.pathname + window.location.search)}`)}>
                Войти
              </Button>
            }
          >
            Для просмотра плана нужен вход в аккаунт.
          </Alert>
        ) : invalidPlanId ? (
          <Alert severity="error">Не передан корректный plan_id.</Alert>
        ) : loading && !status ? (
          <Box sx={{ py: 8, display: "grid", placeItems: "center" }}>
            <CircularProgress />
          </Box>
        ) : (
          <Stack spacing={2.5}>
            {error && <Alert severity="error">{error}</Alert>}

            <Card sx={surfaceCardSx}>
              <CardContent>
                <Stack
                  direction={{ xs: "column", md: "row" }}
                  justifyContent="space-between"
                  alignItems={{ xs: "flex-start", md: "center" }}
                  spacing={1.5}
                >
                  <Stack spacing={0.5}>
                    <Typography sx={{ fontWeight: 900, color: "#0f172a" }}>Статус генерации</Typography>
                    <Typography color="text.secondary">
                      {status ? statusLabel(status.status) : "Загружается..."}
                    </Typography>
                  </Stack>

                  <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap" }}>
                    {status && <Chip label={statusLabel(status.status)} color={statusColor(status.status)} />}
                    {plan && (
                      <Chip label={`Бюджет: ${Math.round(plan.budget).toLocaleString("ru-RU")} ₽`} />
                    )}
                    {plan && <Chip label={`Наборов: ${plan.bundles.length}`} />}
                  </Stack>
                </Stack>

                {status?.status === "queued" || status?.status === "generating" ? (
                  <Box sx={{ mt: 2 }}>
                    <LinearProgress
                      variant={typeof status.progress === "number" ? "determinate" : "indeterminate"}
                      value={typeof status.progress === "number" ? status.progress * 100 : undefined}
                    />
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                      {typeof status.progress === "number"
                        ? `Прогресс: ${(status.progress * 100).toFixed(0)}%`
                        : "Система рассчитывает план, страница обновится автоматически."}
                    </Typography>
                  </Box>
                ) : null}
              </CardContent>
            </Card>

            {resultTabs.length > 0 && (
              <Card sx={surfaceCardSx}>
                <CardContent>
                  <Stack spacing={1.5}>
                    <Box>
                      <Typography sx={{ fontWeight: 900, color: "#0f172a" }}>
                        Результаты по этапам
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Новые результаты расчёта появляются здесь отдельными вкладками.
                      </Typography>
                    </Box>

                    <Tabs
                      value={activeResultTab}
                      onChange={(_, value) => setActiveResultTab(value as ResultTabKey)}
                      variant="scrollable"
                      scrollButtons="auto"
                    >
                      {resultTabs.map((tab) => (
                        <Tab key={tab.key} value={tab.key} label={tab.label} />
                      ))}
                    </Tabs>
                  </Stack>
                </CardContent>
              </Card>
            )}

            {activeResultTab !== "final" ? (
              <StageArtifactPanel artifact={activeStageArtifact} />
            ) : (
              <Stack direction={{ xs: "column", md: "row" }} spacing={2.5}>
              <Card sx={{ ...surfaceCardSx, flex: 1 }}>
                <CardContent>
                  <Typography sx={{ fontWeight: 900, color: "#0f172a", mb: 1.2 }}>
                    Детали плана
                  </Typography>

                  {plan ? (
                    <Stack spacing={2}>
                      <PreviewArea
                        uploadedPlan={uploadedPlan}
                        floorData={simulationFloorData}
                        devices={selectedSimulationDevices}
                      />

                      <Stack direction={{ xs: "column", sm: "row" }} spacing={1} sx={{ flexWrap: "wrap" }}>
                        <Chip label={`Основная экосистема: ${plan.main_ecosystem_id}`} />
                        <Chip label={`Требований: ${plan.requirements.length}`} />
                        <Chip label={`Наборов: ${plan.bundles.length}`} />
                      </Stack>

                      <Box>
                        <Typography sx={{ fontWeight: 800, mb: 1 }}>Требования</Typography>
                        <Stack spacing={1}>
                          {plan.requirements.map((requirement) => (
                            <Box
                              key={requirement.id}
                              sx={{
                                borderRadius: 3,
                                p: 1.5,
                                border: "1px solid rgba(148,163,184,0.18)",
                                background: "#f8fafc",
                              }}
                            >
                              <Typography sx={{ fontWeight: 700 }}>
                                {requirement.device_type} · {requirement.quantity} шт.
                              </Typography>
                              <Typography variant="body2" color="text.secondary">
                                Фильтров: {requirement.filters.length}
                              </Typography>
                            </Box>
                          ))}
                        </Stack>
                      </Box>

                      <Box>
                        <Typography sx={{ fontWeight: 800, mb: 1 }}>
                          Подобранные наборы устройств
                        </Typography>
                        <Stack spacing={1}>
                          {plan.bundles.map((bundle) => (
                            <Box
                              key={bundle.id}
                              onClick={() => {
                                setSelectedBundleId(bundle.id);
                                setSelectedListingId(bundle.listings[0]?.id ?? null);
                              }}
                              sx={{
                                cursor: "pointer",
                                borderRadius: 4,
                                p: 1.8,
                                border:
                                  bundle.id === selectedBundle?.id
                                    ? "2px solid #22c55e"
                                    : "1px solid rgba(148,163,184,0.18)",
                                background: bundle.is_recommended
                                  ? "linear-gradient(135deg, rgba(34,197,94,0.12), rgba(236,253,245,0.8))"
                                  : "#fff",
                              }}
                            >
                              <Stack
                                direction={{ xs: "column", md: "row" }}
                                justifyContent="space-between"
                                spacing={1}
                              >
                                <Box>
                                  <Typography sx={{ fontWeight: 800 }}>Набор #{bundle.id}</Typography>
                                  <Typography variant="body2" color="text.secondary">
                                    Позиций устройств: {simulationDevicesFromBundle(bundle, simulationFloorData).length}
                                  </Typography>
                                </Box>
                                <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap" }}>
                                  {bundle.is_recommended && <Chip color="success" label="Рекомендованный" />}
                                  <Chip label={`Стоимость: ${Math.round(bundle.total_cost).toLocaleString("ru-RU")} ₽`} />
                                  <Chip label={`Качество: ${bundle.quality_score.toFixed(2)}`} />
                                  <Chip label={`Экосистем: ${bundle.extra_ecosystems_used}`} />
                                  <Chip label={`Хабов: ${bundle.hubs_used}`} />
                                </Stack>
                              </Stack>
                            </Box>
                          ))}
                        </Stack>
                      </Box>
                    </Stack>
                  ) : (
                    <Typography color="text.secondary">Ожидаем готовый план.</Typography>
                  )}
                </CardContent>
              </Card>

              <Card sx={{ ...surfaceCardSx, width: { xs: "100%", md: 430 } }}>
                <CardContent>
                  <Typography sx={{ fontWeight: 900, color: "#0f172a", mb: 1 }}>
                    Карточка набора / устройства
                  </Typography>
                  <Divider sx={{ mb: 2 }} />

                  {!selectedBundle ? (
                    <Typography color="text.secondary">Пока нет доступных наборов.</Typography>
                  ) : (
                    <Stack spacing={2}>
                      <Box>
                        <Typography sx={{ fontWeight: 800 }}>Набор #{selectedBundle.id}</Typography>
                        <Typography variant="body2" color="text.secondary">
                          Стоимость: {Math.round(selectedBundle.total_cost).toLocaleString("ru-RU")} ₽ ·
                          качество {selectedBundle.quality_score.toFixed(2)}
                        </Typography>
                      </Box>

                      <Stack spacing={1}>
                        {selectedBundle.listings.map((listing) => (
                          <Stack
                            key={listing.id}
                            direction="row"
                            spacing={1.4}
                            alignItems="center"
                            onClick={() => setSelectedListingId(listing.id)}
                            sx={{
                              p: 1.2,
                              borderRadius: 3,
                              cursor: "pointer",
                              border:
                                listing.id === selectedListing?.id
                                  ? "2px solid #2563eb"
                                  : "1px solid rgba(148,163,184,0.18)",
                              background:
                                listing.id === selectedListing?.id
                                  ? "linear-gradient(135deg, rgba(37,99,235,0.08), rgba(239,246,255,0.9))"
                                  : "#fff",
                            }}
                          >
                            <Avatar
                              src={listing.image_url ?? undefined}
                              alt={listing.name}
                              sx={{ width: 52, height: 52 }}
                            />
                            <Box sx={{ flex: 1 }}>
                              <Typography sx={{ fontWeight: 700, lineHeight: 1.2 }}>
                                {listing.name}
                              </Typography>
                              <Typography variant="body2" color="text.secondary">
                                {listing.device_brand} {listing.device_model}
                              </Typography>
                            </Box>
                            <Typography sx={{ fontWeight: 800 }}>
                              {Math.round(listing.price).toLocaleString("ru-RU")} ₽
                            </Typography>
                          </Stack>
                        ))}
                      </Stack>

                      {selectedSimulationDevices.length > 0 && (
                        <Stack spacing={1}>
                          <Typography sx={{ fontWeight: 800 }}>Устройства на плане</Typography>
                          {selectedSimulationDevices.slice(0, 8).map((device) => (
                            <Box
                              key={device.id}
                              sx={{
                                p: 1.2,
                                borderRadius: 3,
                                background: "#f8fafc",
                                border: "1px solid rgba(148,163,184,0.18)",
                              }}
                            >
                              <Typography sx={{ fontWeight: 700, lineHeight: 1.2 }}>
                                {device.name}
                              </Typography>
                              <Typography variant="body2" color="text.secondary">
                                {device.type} · комната {device.room_id}
                                {device.position ? ` · x ${formatCoordinate(device.position.x)}, y ${formatCoordinate(device.position.y)}` : ""}
                              </Typography>
                            </Box>
                          ))}
                          {selectedSimulationDevices.length > 8 && (
                            <Typography variant="body2" color="text.secondary">
                              И ещё {selectedSimulationDevices.length - 8} устройств.
                            </Typography>
                          )}
                        </Stack>
                      )}

                      {selectedListing && (
                        <>
                          <Divider />
                          <Stack spacing={1.2}>
                            <Typography sx={{ fontWeight: 800 }}>{selectedListing.name}</Typography>
                            <Typography variant="body2" color="text.secondary">
                              {selectedListing.device_brand} {selectedListing.device_model}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                              Купить: {selectedListing.units_to_buy} шт. · В одном листинге: {selectedListing.devices_per_listing} шт.
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                              Quality: {selectedListing.device_quality_score.toFixed(2)}
                            </Typography>

                            <Accordion disableGutters elevation={0} sx={{ borderRadius: 3, border: "1px solid rgba(148,163,184,0.18)" }}>
                              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                                <Typography sx={{ fontWeight: 800 }}>Информация о подключении</Typography>
                              </AccordionSummary>
                              <AccordionDetails>
                                <Stack spacing={1}>
                                  <Typography variant="body2">
                                    Шаг 1: {selectedListing.connection_info.direct_ecosystem} · {selectedListing.connection_info.direct_protocol}
                                  </Typography>
                                  {selectedListing.connection_info.direct_description && (
                                    <Typography variant="body2" color="text.secondary">
                                      {selectedListing.connection_info.direct_description}
                                    </Typography>
                                  )}
                                  {selectedListing.connection_info.final_description && (
                                    <>
                                      <Typography variant="body2">
                                        Шаг 2: {selectedListing.connection_info.final_ecosystem} · {selectedListing.connection_info.final_protocol}
                                      </Typography>
                                      <Typography variant="body2" color="text.secondary">
                                        {selectedListing.connection_info.final_description}
                                      </Typography>
                                    </>
                                  )}
                                </Stack>
                              </AccordionDetails>
                            </Accordion>

                            <Button
                              variant="contained"
                              onClick={() => window.open(selectedListing.url, "_blank", "noopener,noreferrer")}
                              sx={{ fontWeight: 900, borderRadius: 3 }}
                            >
                              Открыть товар
                            </Button>
                          </Stack>
                        </>
                      )}

                      <Typography variant="body2" color="text.secondary">
                        Сейчас выбрано устройств в наборе: {bundleTotalListings}
                      </Typography>

                      <Button
                        variant="outlined"
                        disabled={!selectedBundle.listings.length}
                        onClick={() => openSimulation(selectedBundle, simulationFloorData)}
                        sx={{ fontWeight: 900, borderRadius: 3 }}
                      >
                        Открыть в симуляции
                      </Button>
                    </Stack>
                  )}
                </CardContent>
              </Card>
              </Stack>
            )}
          </Stack>
        )}
      </Box>
    </Box>
  );
}

function PreviewArea(props: { uploadedPlan: UploadedPlanState | null; floorData?: unknown; devices?: unknown[] }) {
  const floorFromStages = normalizeFloorPreviewData(props.floorData);
  if (floorFromStages.floor) {
    return (
      <Box
        sx={{
          position: "relative",
          width: "100%",
          aspectRatio: "16 / 10",
          borderRadius: 4,
          overflow: "hidden",
          border: "1px solid rgba(148,163,184,0.24)",
          background: "#f4f6f8",
        }}
      >
        <ApartmentPlanPreview floor={floorFromStages.floor} devices={props.devices} zones={floorFromStages.zones} />
      </Box>
    );
  }

  if (!props.uploadedPlan) {
    return (
      <Box
        sx={{
          position: "relative",
          width: "100%",
          aspectRatio: "16 / 10",
          borderRadius: 4,
          overflow: "hidden",
          border: "1px solid rgba(148,163,184,0.24)",
        }}
      >
        <Image src="/floorplan.png" alt="Floor plan" fill style={{ objectFit: "cover" }} unoptimized />
      </Box>
    );
  }

  if (props.uploadedPlan.planDataUrl) {
    return (
      <Box
        sx={{
          position: "relative",
          width: "100%",
          aspectRatio: "16 / 10",
          borderRadius: 4,
          overflow: "hidden",
          border: "1px solid rgba(148,163,184,0.24)",
        }}
      >
        <Image src={props.uploadedPlan.planDataUrl} alt="Uploaded plan" fill style={{ objectFit: "contain" }} unoptimized />
      </Box>
    );
  }

  const floor = props.uploadedPlan.floorJson ?? props.uploadedPlan.parsedFloor ?? props.uploadedPlan.floor;
  if (floor) {
    return (
      <Box
        sx={{
          position: "relative",
          width: "100%",
          aspectRatio: "16 / 10",
          borderRadius: 4,
          overflow: "hidden",
          border: "1px solid rgba(148,163,184,0.24)",
          background: "#f4f6f8",
        }}
      >
        <ApartmentPlanPreview floor={floor} devices={props.devices} zones={props.uploadedPlan.zones} />
      </Box>
    );
  }

  return (
    <Box
      sx={{
        width: "100%",
        aspectRatio: "16 / 10",
        borderRadius: 4,
        border: "1px solid rgba(148,163,184,0.24)",
        display: "grid",
        placeItems: "center",
        px: 3,
        background: "linear-gradient(135deg, rgba(37,99,235,0.08), rgba(15,23,42,0.04))",
        textAlign: "center",
      }}
    >
      <Stack spacing={1}>
        <Typography sx={{ fontWeight: 800, color: "#1e293b" }}>DXF-файл загружен</Typography>
        <Typography color="text.secondary">{props.uploadedPlan.fileName}</Typography>
        <Typography variant="body2" color="text.secondary">
          Для DXF пока показываем только факт загрузки.
        </Typography>
      </Stack>
    </Box>
  );
}

function collectStageArtifacts(
  status: ApiPlanStatus | null,
  plan: ApiHomePlan | null
): ApiPlanStageArtifact[] {
  const byKey = new Map<string, ApiPlanStageArtifact>();

  for (const artifact of status?.stages ?? []) {
    if (artifact.key) byKey.set(artifact.key, artifact);
  }

  for (const artifact of [...(plan?.stages ?? []), ...(plan?.artifacts ?? [])]) {
    if (artifact.key) byKey.set(artifact.key, artifact);
  }

  return Array.from(byKey.values());
}

function buildResultTabs(stageArtifacts: ApiPlanStageArtifact[], hasFinalPlan: boolean): ResultTab[] {
  const tabs: ResultTab[] = [];
  const zones = findStageArtifact(stageArtifacts, ["zones", "zoning", "rooms", "layout_zones"]);
  const energy = findStageArtifact(stageArtifacts, ["energy", "energy_report", "power", "consumption"]);
  const floor = findStageArtifact(stageArtifacts, ["parsed_floor_plan", "floor_plan", "floor", "apartment"]);
  const layout = findStageArtifact(stageArtifacts, ["layout", "placement", "placements"]);
  const deviceSelection = findStageArtifact(stageArtifacts, ["device_selection", "pareto", "bundle", "bundles"]);
  const promoted = new Set([zones, energy, floor, layout, deviceSelection].filter(Boolean));
  const extraStages = stageArtifacts.filter((artifact) => !promoted.has(artifact));

  if (hasFinalPlan) {
    tabs.push({ key: "final", label: "Финал" });
  }

  if (zones) {
    tabs.push({ key: "zones", label: "Зоны", artifact: zones });
  }

  if (energy) {
    tabs.push({ key: "energy", label: "Энергопотребление", artifact: energy });
  }

  if (floor) {
    tabs.push({ key: "parsed_floor_plan", label: "Квартира", artifact: floor });
  }

  if (layout) {
    tabs.push({ key: "layout", label: "Расстановка", artifact: layout });
  }

  if (deviceSelection) {
    tabs.push({ key: "device_selection", label: "Подбор устройств", artifact: deviceSelection });
  }

  for (const artifact of extraStages) {
    tabs.push({
      key: artifact.key,
      label: artifact.title ?? artifact.name ?? artifact.key,
      artifact,
    });
  }

  return tabs;
}

function findStageArtifact(stageArtifacts: ApiPlanStageArtifact[], keys: string[]) {
  return stageArtifacts.find((artifact) => {
    const normalizedKey = artifact.key.toLowerCase();
    const normalizedTitle = `${artifact.name ?? ""} ${artifact.title ?? ""}`.toLowerCase();
    return keys.some((key) => normalizedKey.includes(key) || normalizedTitle.includes(key));
  });
}

function StageArtifactPanel(props: { artifact?: ApiPlanStageArtifact }) {
  if (!props.artifact) {
    return (
      <Card sx={surfaceCardSx}>
        <CardContent>
          <Typography color="text.secondary">Данные этого этапа пока не готовы.</Typography>
        </CardContent>
      </Card>
    );
  }

  const payload = props.artifact.data ?? props.artifact.payload ?? props.artifact;
  const floorPreview = normalizeFloorPreviewData(payload);
  const bundles = normalizeDeviceSelectionBundles(payload);
  const layoutDevices = devicesFromLayout(payload);

  return (
    <Card sx={surfaceCardSx}>
      <CardContent>
        <Stack spacing={2}>
          <Box>
            <Stack direction="row" spacing={1} alignItems="center" sx={{ flexWrap: "wrap", mb: 0.5 }}>
              <Typography sx={{ fontWeight: 900, color: "#0f172a" }}>
                {props.artifact.title ?? props.artifact.name ?? props.artifact.key}
              </Typography>
              {props.artifact.status && <Chip size="small" label={props.artifact.status} />}
            </Stack>
            <Typography variant="body2" color="text.secondary">
              Артефакт этапа сохранён на фронте и не пропадает после получения финального результата.
            </Typography>
          </Box>

          {typeof props.artifact.progress === "number" && (
            <Box>
              <LinearProgress variant="determinate" value={props.artifact.progress * 100} />
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.8 }}>
                Прогресс этапа: {(props.artifact.progress * 100).toFixed(0)}%
              </Typography>
            </Box>
          )}

          {Boolean(floorPreview.floor) && (
            <Box
              sx={{
                position: "relative",
                width: "100%",
                aspectRatio: "16 / 10",
                borderRadius: 4,
                overflow: "hidden",
                border: "1px solid rgba(148,163,184,0.24)",
                background: "#f4f6f8",
              }}
            >
              <ApartmentPlanPreview
                floor={floorPreview.floor}
                devices={bundles[0] ? simulationDevicesFromBundle(bundles[0], payload) : layoutDevices}
                zones={floorPreview.zones}
              />
            </Box>
          )}

          {layoutDevices.length > 0 && <LayoutDevicesSummary devices={layoutDevices} />}

          {bundles.length > 0 && <DeviceSelectionSummary bundles={bundles} />}

          {!floorPreview.floor && !layoutDevices.length && !bundles.length && (
            <Alert severity="info">Этап готов, но для него пока нет отдельного визуального представления.</Alert>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}

function LayoutDevicesSummary(props: { devices: SimulationDevice[] }) {
  return (
    <Stack spacing={1.2}>
      <Typography sx={{ fontWeight: 900, color: "#0f172a" }}>
        Расставленные устройства
      </Typography>
      <Stack spacing={1}>
        {props.devices.map((device) => (
          <Box
            key={device.id}
            sx={{
              borderRadius: 3,
              p: 1.5,
              background: "#f8fafc",
              border: "1px solid rgba(148,163,184,0.22)",
            }}
          >
            <Stack direction={{ xs: "column", sm: "row" }} spacing={1} justifyContent="space-between">
              <Box>
                <Typography sx={{ fontWeight: 800 }}>{device.name}</Typography>
                <Typography variant="body2" color="text.secondary">
                  {device.type} · комната {device.room_id}
                </Typography>
              </Box>
              {device.position && (
                <Chip
                  size="small"
                  label={`x ${formatCoordinate(device.position.x)}, y ${formatCoordinate(device.position.y)}`}
                />
              )}
            </Stack>
          </Box>
        ))}
      </Stack>
    </Stack>
  );
}

function DeviceSelectionSummary(props: { bundles: ApiHomePlan["bundles"] }) {
  return (
    <Stack spacing={1.2}>
      <Typography sx={{ fontWeight: 900, color: "#0f172a" }}>
        Наборы из device_selection
      </Typography>
      {props.bundles.map((bundle) => (
        <Box
          key={bundle.id}
          sx={{
            borderRadius: 3,
            p: 1.5,
            background: bundle.is_recommended ? "#ecfdf5" : "#f8fafc",
            border: "1px solid rgba(148,163,184,0.22)",
          }}
        >
          <Stack spacing={0.8}>
            <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap" }}>
              <Chip size="small" label={`Набор #${bundle.id}`} />
              {bundle.is_recommended && <Chip size="small" color="success" label="Рекомендованный" />}
              <Chip size="small" label={`${Math.round(bundle.total_cost).toLocaleString("ru-RU")} ₽`} />
              <Chip size="small" label={`Качество ${bundle.quality_score.toFixed(2)}`} />
              <Chip size="small" label={`Устройств ${bundle.listings.length}`} />
            </Stack>
            <Stack spacing={0.5}>
              {bundle.listings.slice(0, 5).map((listing) => (
                <Typography key={listing.id} variant="body2" color="text.secondary">
                  {listing.name} · {listing.units_to_buy} шт.
                </Typography>
              ))}
            </Stack>
          </Stack>
        </Box>
      ))}
    </Stack>
  );
}

function statusLabel(status: ApiPlanStatus["status"]) {
  switch (status) {
    case "queued":
      return "В очереди";
    case "generating":
      return "Генерируется";
    case "completed":
      return "Завершён";
    case "failed":
      return "Ошибка";
    default:
      return "Неизвестно";
  }
}

function statusColor(status: ApiPlanStatus["status"]): "default" | "primary" | "success" | "error" | "warning" {
  switch (status) {
    case "queued":
      return "warning";
    case "generating":
      return "primary";
    case "completed":
      return "success";
    case "failed":
      return "error";
    default:
      return "default";
  }
}

const surfaceCardSx = {
  borderRadius: 5,
  background: "rgba(255,255,255,0.96)",
  boxShadow: "0 26px 70px rgba(15,23,42,0.22)",
  border: "1px solid rgba(226,232,240,0.8)",
};

function loadUploadedPlan(): UploadedPlanState | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem("planner-uploaded-plan");
    return raw ? (JSON.parse(raw) as UploadedPlanState) : null;
  } catch {
    return null;
  }
}

type SimulationBundle = NonNullable<ApiHomePlan["bundles"][number]>;
const SIMULATION_RETURN_STORAGE_KEY = "simulation-return-url";

function simulationUrl() {
  return process.env.NEXT_PUBLIC_SIM_UI_URL ?? "http://127.0.0.1:3001/simulation";
}

function openSimulationFromPlan(planId: number | string, floor?: unknown) {
  if (floor) {
    localStorage.setItem("simulation-floor", JSON.stringify(floor));
  }

  const url = new URL(simulationUrl(), window.location.origin);
  localStorage.setItem(SIMULATION_RETURN_STORAGE_KEY, window.location.href);
  if (typeof planId === "number" && Number.isFinite(planId) && planId > 0) {
    url.searchParams.set("plan_id", String(planId));
  } else if (typeof planId === "string" && planId) {
    url.searchParams.set("workflow_id", planId);
  }
  url.searchParams.set("returnTo", window.location.href);
  window.location.href = url.toString();
}

function openSimulation(bundle: SimulationBundle, floor?: unknown) {
  const devices = simulationDevicesFromBundle(bundle, floor);
  const triggerIds = triggerDeviceIdsFromDevices(devices);

  localStorage.setItem("simulation-devices", JSON.stringify(devices));
  localStorage.setItem("simulation-trigger-device-ids", JSON.stringify(triggerIds));
  if (floor) {
    localStorage.setItem("simulation-floor", JSON.stringify(floor));
  }

  const url = new URL(simulationUrl(), window.location.origin);
  localStorage.setItem(SIMULATION_RETURN_STORAGE_KEY, window.location.href);
  url.searchParams.set("returnTo", window.location.href);
  url.searchParams.set("devices", JSON.stringify(devices));
  if (triggerIds.length) {
    url.searchParams.set("trigger_ids", triggerIds.join(","));
  }
  window.location.href = url.toString();
}

function simulationDevicesFromBundle(bundle: SimulationBundle, floor?: unknown): SimulationDevice[] {
  const layoutDevices = devicesFromLayout(floor);
  const usedLayoutIndexes = new Set<number>();
  const result: SimulationDevice[] = [];

  bundle.listings.forEach((listing, listingIndex) => {
    const type = listing.device_attributes?.device_type;
    const normalizedType = typeof type === "string" ? type : listing.name;
    const units = Math.max(1, listing.units_to_buy || 1);

    for (let unitIndex = 0; unitIndex < units; unitIndex += 1) {
      const matchedIndex = layoutDevices.findIndex((device, index) => {
        if (usedLayoutIndexes.has(index)) return false;
        return devicesMatchListing(device, listing, normalizedType);
      });
      const matched = matchedIndex >= 0 ? layoutDevices[matchedIndex] : null;
      if (matchedIndex >= 0) usedLayoutIndexes.add(matchedIndex);

      const id = matched?.id ?? makeSimulationDeviceId(listing, listingIndex, unitIndex);
      result.push({
        id,
        trigger_id: id,
        name: listing.name,
        type: matched?.type ?? normalizedType,
        device_type: matched?.device_type ?? normalizedType,
        room_id: matched?.room_id ?? roomIdForDevice(normalizedType, listingIndex + unitIndex),
        position: matched?.position,
        direction: matched?.direction,
        track: matched?.track,
        filters: matched?.filters,
        listing_id: listing.id,
        requirement_id: listing.requirement_id,
      });
    }
  });

  for (const [index, device] of layoutDevices.entries()) {
    if (!usedLayoutIndexes.has(index)) result.push(device);
  }

  return result;
}

function devicesFromLayout(value: unknown): SimulationDevice[] {
  const layout = findLayoutPayload(value);
  const placements = asRecord(layout)?.placements;
  if (!placements || typeof placements !== "object" || Array.isArray(placements)) return [];

  return Object.entries(placements as Record<string, unknown>).flatMap(([roomId, rawPlacements]) => {
    const placementList = asArray(rawPlacements) ?? [];
    return placementList.flatMap((placement, index) => normalizeLayoutPlacement(placement, roomId, index));
  });
}

function normalizeLayoutPlacement(value: unknown, roomId: string, index: number): SimulationDevice[] {
  const placement = asRecord(value);
  if (!placement) return [];

  const device = asRecord(placement.device);
  const type = toText(device?.type ?? device?.name ?? placement.device_type ?? placement.type, "device");
  const id = toText(device?.id ?? placement.id, `${type}_${roomId}_${index + 1}`);
  const position = normalizePoint(placement.position);
  const direction = normalizePoint(placement.direction);
  const filters = asRecord(placement.filters) ?? undefined;

  return [
    {
      id,
      trigger_id: id,
      name: humanizeDeviceType(type),
      type,
      device_type: type,
      room_id: roomId,
      position,
      direction,
      track: toNullableText(device?.track ?? placement.track) ?? undefined,
      filters,
    },
  ];
}

function findLayoutPayload(value: unknown): unknown {
  if (!value || typeof value !== "object") return null;
  const record = value as Record<string, unknown>;
  if (record.placements && typeof record.placements === "object") return value;

  for (const key of ["layout", "device_layout", "placements"]) {
    const nested = record[key];
    if (!nested) continue;
    const match = findLayoutPayload(nested);
    if (match) return match;
  }

  for (const nested of Object.values(record)) {
    const match = findLayoutPayload(nested);
    if (match) return match;
  }

  return null;
}

function normalizePoint(value: unknown): { x: number; y: number } | undefined {
  const point = asRecord(value);
  if (!point) return undefined;
  const x = toNumber(point.x, NaN);
  const y = toNumber(point.y, NaN);
  return Number.isFinite(x) && Number.isFinite(y) ? { x, y } : undefined;
}

function devicesMatchListing(
  device: SimulationDevice,
  listing: ApiHomePlan["bundles"][number]["listings"][number],
  fallbackType: string
) {
  const listingType = normalizeDeviceKey(fallbackType);
  const deviceType = normalizeDeviceKey(device.device_type);
  const listingName = normalizeDeviceKey(`${listing.name} ${listing.device_brand} ${listing.device_model}`);
  const attributes = normalizeDeviceKey(Object.values(listing.device_attributes ?? {}).join(" "));

  return (
    listingType === deviceType ||
    listingType.includes(deviceType) ||
    deviceType.includes(listingType) ||
    listingName.includes(deviceType) ||
    attributes.includes(deviceType)
  );
}

function normalizeDeviceKey(value: unknown) {
  return toText(value, "")
    .toLowerCase()
    .replace(/[^a-z0-9а-яё]+/gi, "_")
    .replace(/^_+|_+$/g, "");
}

function humanizeDeviceType(type: string) {
  return type
    .split("_")
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(" ");
}

function formatCoordinate(value: number) {
  return Number.isInteger(value) ? String(value) : value.toFixed(2);
}

function triggerDeviceIdsFromDevices(devices: Array<{ id: string; type?: string; device_type?: string }>) {
  return devices
    .filter((device) => isTriggerDeviceType(device.type ?? device.device_type ?? device.id))
    .map((device) => device.id);
}

function isTriggerDeviceType(value: string) {
  const key = value.toLowerCase();
  return (
    key.includes("motion") ||
    key.includes("presence") ||
    key.includes("sensor") ||
    key.includes("button") ||
    key.includes("switch") ||
    key.includes("door") ||
    key.includes("window") ||
    key.includes("leak") ||
    key.includes("gas") ||
    key.includes("smoke")
  );
}

function roomIdForDevice(type: string, index: number) {
  const key = type.toLowerCase();
  if (key.includes("leak") || key.includes("water")) return "bath";
  if (key.includes("gas") || key.includes("smoke")) return "kitchen";
  if (key.includes("door") || key.includes("motion") || key.includes("presence")) return "hall";
  if (key.includes("temperature") || key.includes("climate")) return "living";
  return ["living", "hall", "kitchen", "bath"][index % 4];
}

function collectSimulationFloorData(
  uploadedPlan: UploadedPlanState | null,
  status: ApiPlanStatus | null,
  plan: ApiHomePlan | null
) {
  const fromUpload = uploadedPlan?.floorJson ?? uploadedPlan?.parsedFloor ?? uploadedPlan?.floor;
  let floor: unknown = fromUpload ?? null;
  let zones: unknown = null;
  let layout: unknown = null;

  const artifacts = collectStageArtifacts(status, plan);
  for (const artifact of artifacts) {
    const payload = artifact.data ?? artifact.payload;
    floor ??= findFloorPayload(payload);
    zones ??= findPayloadByKeys(payload, ["zones", "zone"]);
    layout ??= findLayoutPayload(payload);
  }

  if (!floor && !zones && !layout) return null;
  if (floor && !zones && !layout) return floor;

  return {
    floor,
    zones,
    layout,
  };
}

function findFloorPayload(value: unknown): unknown {
  if (!value || typeof value !== "object") return null;

  const record = value as Record<string, unknown>;
  if (Array.isArray(record.walls) || Array.isArray(record.rooms) || record.layout || record.apartment) {
    return value;
  }

  for (const key of ["floor", "floor_plan", "floorJson", "parsedFloor", "parsed_floor_plan", "apartment", "layout", "plan"]) {
    const nested = record[key];
    if (!nested) continue;
    const match = findFloorPayload(nested);
    if (match) return match;
  }

  return null;
}

function findPayloadByKeys(value: unknown, keys: string[]): unknown {
  if (!value || typeof value !== "object") return null;
  const record = value as Record<string, unknown>;

  for (const [key, nested] of Object.entries(record)) {
    const normalized = key.toLowerCase();
    if (keys.some((candidate) => normalized === candidate || normalized.includes(candidate))) {
      return nested;
    }
  }

  for (const nested of Object.values(record)) {
    const match = findPayloadByKeys(nested, keys);
    if (match) return match;
  }

  return null;
}

function isPipelinePending(value: ApiPipelineResult | PipelineStatusResponse): value is PipelineStatusResponse {
  return "status" in value && typeof value.status === "string" && !("parsed_floor_plan" in value);
}

function normalizePipelineStatus(status: string): ApiPlanStatus["status"] {
  const normalized = status.toLowerCase();
  if (normalized.includes("failed") || normalized.includes("terminated") || normalized.includes("canceled") || normalized.includes("timed_out")) {
    return "failed";
  }
  if (normalized.includes("completed")) {
    return "completed";
  }
  if (normalized.includes("queued")) {
    return "queued";
  }
  return "generating";
}

function normalizeFloorPreviewData(value: unknown): { floor: unknown | null; zones?: unknown } {
  const floor = findFloorPayload(value);
  const zones = findPayloadByKeys(value, ["zones", "zone"]);
  return { floor: floor ?? null, zones: zones ?? undefined };
}

function normalizeDeviceSelectionBundles(value: unknown): ApiHomePlan["bundles"] {
  const record = asRecord(value);
  if (!record) return [];

  const source = asRecord(record.device_selection) ?? record;
  const directBundles = asArray(source.bundles) ?? asArray(source.solutions);
  if (directBundles) {
    return directBundles.map((bundle, index) => normalizeBundle(bundle, index)).filter(Boolean) as ApiHomePlan["bundles"];
  }

  const paretoPoints = asArray(source.pareto_points) ?? asArray(source.points) ?? asArray(source.result);
  if (!paretoPoints) return [];

  return paretoPoints.map((point, index) => normalizeBundle(point, index)).filter(Boolean) as ApiHomePlan["bundles"];
}

function normalizeBundle(value: unknown, index: number): ApiHomePlan["bundles"][number] | null {
  const record = asRecord(value);
  if (!record) return null;

  const rawItems = asArray(record.listings) ?? asArray(record.items) ?? asArray(record.devices) ?? [];
  const listings = rawItems
    .flatMap((item, itemIndex) => normalizeListingsFromItem(item, index, itemIndex))
    .filter(Boolean) as ApiHomePlan["bundles"][number]["listings"];

  return {
    id: toNumber(record.id, index + 1),
    total_cost: toNumber(record.total_cost ?? record.cost, listings.reduce((sum, listing) => sum + listing.price * listing.units_to_buy, 0)),
    quality_score: toNumber(record.quality_score ?? record.avg_quality ?? record.quality, 0),
    extra_ecosystems_used: toNumber(record.extra_ecosystems_used ?? record.num_ecosystems, 0),
    hubs_used: toNumber(record.hubs_used ?? record.num_hubs, 0),
    is_recommended: Boolean(record.is_recommended ?? index === 0),
    ecosystems_used: asStringArray(record.ecosystems_used),
    listings,
  };
}

function normalizeListingsFromItem(
  value: unknown,
  bundleIndex: number,
  itemIndex: number
): ApiHomePlan["bundles"][number]["listings"] {
  const record = asRecord(value);
  if (!record) return [];

  if (record.best_listing || record.listings) {
    const bestListing = asRecord(record.best_listing);
    const listings = asArray(record.listings);
    const listingSource = bestListing ?? asRecord(listings?.[0]) ?? record;
    return [normalizeListing(listingSource, record, bundleIndex, itemIndex)];
  }

  return [normalizeListing(record, record, bundleIndex, itemIndex)];
}

function normalizeListing(
  source: Record<string, unknown>,
  item: Record<string, unknown>,
  bundleIndex: number,
  itemIndex: number
): ApiHomePlan["bundles"][number]["listings"][number] {
  const sourceId = source.id ?? source.listing_id ?? source.source_listing_id ?? item.device_id;
  const id = toNumber(sourceId, (bundleIndex + 1) * 1000 + itemIndex + 1);
  const brand = toText(source.brand ?? item.brand, "Неизвестный бренд");
  const model = toText(source.model ?? item.model, "");
  const category = toText(item.category ?? item.device_type ?? item.type, "Устройство");
  const name = toText(source.name ?? source.title, [brand, model].filter(Boolean).join(" ") || category);
  const price = toNumber(source.price ?? source.price_each ?? item.price_each ?? item.price, 0);
  const quantity = toNumber(item.quantity ?? item.count ?? source.quantity, 1);
  const connection = asRecord(item.connection) ?? asRecord(source.connection_info);

  return {
    id,
    name,
    device_brand: brand,
    device_model: model,
    device_quality_score: toNumber(item.quality ?? source.quality ?? source.device_quality_score, 0),
    price,
    url: toText(source.url ?? source.product_url, "#"),
    image_url: toNullableText(source.image_url ?? item.image_url),
    devices_per_listing: toNumber(source.devices_per_listing, 1),
    units_to_buy: quantity,
    requirement_id: toNumber(item.requirement_id ?? source.requirement_id, itemIndex + 1),
    device_attributes: (asRecord(item.device_attributes) ?? asRecord(source.device_attributes) ?? { device_type: category }) as Record<string, unknown>,
    connection_info: {
      direct_ecosystem: toText(connection?.ecosystem ?? connection?.direct_ecosystem ?? connection?.bridge_ecosystem, ""),
      direct_protocol: toText(connection?.protocol ?? connection?.direct_protocol ?? connection?.method, ""),
      direct_description: toNullableText(connection?.method),
      final_ecosystem: toText(connection?.final_ecosystem ?? connection?.ecosystem ?? connection?.bridge_ecosystem, ""),
      final_protocol: toText(connection?.final_protocol ?? connection?.protocol ?? connection?.method, ""),
      final_description: toNullableText(connection?.bridge_ecosystem),
    },
  };
}

function pipelineResultToStageArtifacts(result: ApiPipelineResult | PipelineStatusResponse): ApiPlanStageArtifact[] {
  const stages: ApiPlanStageArtifact[] = [];
  for (const artifact of [...(result.stages ?? []), ...(result.artifacts ?? [])]) {
    if (artifact.key) stages.push(artifact);
  }
  if (result.parsed_floor_plan) {
    stages.push({ key: "parsed_floor_plan", title: "Распознанный план", status: "completed", payload: result.parsed_floor_plan });
  }
  if (result.layout) {
    stages.push({ key: "layout", title: "Расстановка устройств", status: "completed", payload: result.layout });
  }
  if (result.device_selection) {
    stages.push({ key: "device_selection", title: "Подбор устройств", status: "completed", payload: result.device_selection });
  }
  return stages;
}

function pipelineResultToPlan(result: ApiPipelineResult | PipelineStatusResponse, budget: number): ApiHomePlan {
  const bundles = normalizeDeviceSelectionBundles(result.device_selection ?? result);
  const effectiveBundles = bundles.length ? bundles : layoutDevicesToBundles(devicesFromLayout(result));
  const requirements = collectRequirementsFromBundles(effectiveBundles);

  return {
    plan_id: 0,
    budget,
    main_ecosystem_id: "",
    requirements,
    bundles: effectiveBundles,
    stages: pipelineResultToStageArtifacts(result),
    artifacts: pipelineResultToStageArtifacts(result),
  };
}

function layoutDevicesToBundles(devices: SimulationDevice[]): ApiHomePlan["bundles"] {
  if (!devices.length) return [];

  return [
    {
      id: 1,
      total_cost: 0,
      quality_score: 0,
      extra_ecosystems_used: 0,
      hubs_used: 0,
      is_recommended: true,
      listings: devices.map((device, index) => ({
        id: index + 1,
        name: device.name,
        device_brand: "Layout",
        device_model: device.type,
        device_quality_score: 0,
        price: 0,
        url: "#",
        image_url: null,
        devices_per_listing: 1,
        units_to_buy: 1,
        requirement_id: index + 1,
        device_attributes: { device_type: device.type },
        connection_info: {
          direct_ecosystem: "",
          direct_protocol: "",
          direct_description: null,
          final_ecosystem: "",
          final_protocol: "",
          final_description: null,
        },
      })),
    },
  ];
}

function collectRequirementsFromBundles(bundles: ApiHomePlan["bundles"]): ApiHomePlan["requirements"] {
  const byId = new Map<number, ApiHomePlan["requirements"][number]>();
  for (const bundle of bundles) {
    for (const listing of bundle.listings) {
      if (!byId.has(listing.requirement_id)) {
        byId.set(listing.requirement_id, {
          id: listing.requirement_id,
          device_type: toText(listing.device_attributes?.device_type, listing.name),
          quantity: listing.units_to_buy,
          filters: [],
        });
      }
    }
  }
  return Array.from(byId.values());
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : null;
}

function asArray(value: unknown): unknown[] | null {
  return Array.isArray(value) ? value : null;
}

function asStringArray(value: unknown): string[] | undefined {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : undefined;
}

function toNumber(value: unknown, fallback: number): number {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return fallback;
}

function toText(value: unknown, fallback: string): string {
  return typeof value === "string" && value.trim() ? value.trim() : fallback;
}

function toNullableText(value: unknown): string | null {
  return typeof value === "string" && value.trim() ? value.trim() : null;
}

function budgetFromStorage() {
  if (typeof window === "undefined") return "";
  try {
    const raw = localStorage.getItem("planner-last-budget");
    return raw ?? "";
  } catch {
    return "";
  }
}

function makeSimulationDeviceId(
  listing: ApiHomePlan["bundles"][number]["listings"][number],
  index: number,
  unitIndex = 0
) {
  const rawType = listing.device_attributes?.device_type;
  const type = typeof rawType === "string" && rawType.trim() ? rawType : listing.name;
  const safeType = type.toLowerCase().replace(/[^a-z0-9а-яё]+/gi, "_").replace(/^_+|_+$/g, "");
  return `${safeType || "device"}_${listing.id}_${index + 1}_${unitIndex + 1}`;
}
