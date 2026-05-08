"use client";

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
  Typography,
} from "@mui/material";
import Image from "next/image";
import { api } from "../lib/api";
import type { ApiHomePlan, ApiPlanStatus } from "../lib/types";

type UploadedPlanState = {
  fileName?: string;
  planDataUrl?: string;
  planFileType?: "dxf" | "png" | "";
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
  const router = useRouter();
  const searchParams = useSearchParams();

  const [status, setStatus] = useState<ApiPlanStatus | null>(null);
  const [plan, setPlan] = useState<ApiHomePlan | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [uploadedPlan, setUploadedPlan] = useState<UploadedPlanState | null>(null);
  const [selectedBundleId, setSelectedBundleId] = useState<number | null>(null);
  const [selectedListingId, setSelectedListingId] = useState<number | null>(null);

  const planId = Number(searchParams.get("id") ?? "");

  useEffect(() => {
    try {
      const raw = localStorage.getItem("planner-uploaded-plan");
      if (!raw) return;
      setUploadedPlan(JSON.parse(raw) as UploadedPlanState);
    } catch {
      setUploadedPlan(null);
    }
  }, []);

  useEffect(() => {
    if (!Number.isFinite(planId) || planId <= 0) {
      setError("Не передан корректный plan_id.");
      setLoading(false);
      return;
    }

    let active = true;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const loadStatus = async () => {
      try {
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
  }, [planId]);

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

  const bundleTotalListings = selectedBundle?.listings.length ?? 0;

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
              <Typography variant="h4" sx={{ fontWeight: 900, letterSpacing: "-0.04em", mb: 0.8 }}>
                План умного дома #{Number.isFinite(planId) ? planId : "—"}
              </Typography>
              <Typography sx={{ color: "rgba(255,255,255,0.72)", maxWidth: 620 }}>
                Эта страница работает с backend: статус берётся из
                {" /api/v1/plans/{plan_id}/status, "}
                а готовый результат — из {" /api/v1/plans/{plan_id}."}
              </Typography>
            </Box>

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
        </Box>

        {error ? (
          <Alert severity="error">{error}</Alert>
        ) : loading && !status ? (
          <Box sx={{ py: 8, display: "grid", placeItems: "center" }}>
            <CircularProgress />
          </Box>
        ) : (
          <Stack spacing={2.5}>
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
                        : "Backend ещё рассчитывает план, страница обновится автоматически."}
                    </Typography>
                  </Box>
                ) : null}
              </CardContent>
            </Card>

            <Stack direction={{ xs: "column", md: "row" }} spacing={2.5}>
              <Card sx={{ ...surfaceCardSx, flex: 1 }}>
                <CardContent>
                  <Typography sx={{ fontWeight: 900, color: "#0f172a", mb: 1.2 }}>
                    Детали плана
                  </Typography>

                  {plan ? (
                    <Stack spacing={2}>
                      <PreviewArea uploadedPlan={uploadedPlan} />

                      <Stack direction={{ xs: "column", sm: "row" }} spacing={1} sx={{ flexWrap: "wrap" }}>
                        <Chip label={`Main ecosystem: ${plan.main_ecosystem_id}`} />
                        <Chip label={`Требований: ${plan.requirements.length}`} />
                        <Chip label={`Bundles: ${plan.bundles.length}`} />
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
                                    Устройств: {bundle.listings.length}
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
                    <Typography color="text.secondary">Ожидаем готовый план от backend.</Typography>
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
                    </Stack>
                  )}
                </CardContent>
              </Card>
            </Stack>
          </Stack>
        )}
      </Box>
    </Box>
  );
}

function PreviewArea(props: { uploadedPlan: UploadedPlanState | null }) {
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
          Для DXF пока показываем только факт загрузки, а сам подбор идёт через backend API.
        </Typography>
      </Stack>
    </Box>
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
