"use client";

import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import ArrowForwardRoundedIcon from "@mui/icons-material/ArrowForwardRounded";
import AutoAwesomeRoundedIcon from "@mui/icons-material/AutoAwesomeRounded";
import ChecklistRoundedIcon from "@mui/icons-material/ChecklistRounded";
import HomeWorkRoundedIcon from "@mui/icons-material/HomeWorkRounded";
import TimelineRoundedIcon from "@mui/icons-material/TimelineRounded";
import { Alert, Box, Button, Chip, CircularProgress, Stack, Typography } from "@mui/material";
import Link from "next/link";
import { api } from "./lib/api";
import type { ApiPlanSummary } from "./lib/types";

const steps = [
  "Загрузите план квартиры и выберите основную экосистему.",
  "Соберите требования к устройствам и запустите генерацию плана.",
  "Следите за прогрессом и просматривайте реальные наборы устройств из backend.",
];

export default function Home() {
  const [plans, setPlans] = useState<ApiPlanSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;

    api
      .listPlans()
      .then((response) => {
        if (!active) return;
        setPlans(response);
      })
      .catch((err: unknown) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Не удалось загрузить список планов.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, []);

  const sortedPlans = useMemo(
    () =>
      [...plans].sort(
        (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      ),
    [plans]
  );

  return (
    <Box
      sx={{
        minHeight: "100vh",
        px: { xs: 2, md: 5 },
        py: { xs: 4, md: 7 },
        display: "flex",
        alignItems: "center",
      }}
    >
      <Box sx={{ maxWidth: 1180, mx: "auto", width: "100%" }}>
        <Box
          sx={{
            position: "relative",
            overflow: "hidden",
            borderRadius: 8,
            px: { xs: 3, md: 6 },
            py: { xs: 4, md: 6 },
            background:
              "linear-gradient(145deg, rgba(255,255,255,0.96) 0%, rgba(238,244,255,0.94) 55%, rgba(222,237,255,0.92) 100%)",
            boxShadow: "0 30px 80px rgba(15, 23, 42, 0.16)",
            border: "1px solid rgba(148, 163, 184, 0.22)",
          }}
        >
          <Box
            sx={{
              position: "absolute",
              top: -120,
              right: -80,
              width: 280,
              height: 280,
              borderRadius: "50%",
              background: "radial-gradient(circle, rgba(34,197,94,0.22), rgba(34,197,94,0))",
            }}
          />

          <Stack spacing={4} sx={{ position: "relative" }}>
            <Stack spacing={2}>
              <Chip
                icon={<AutoAwesomeRoundedIcon />}
                label="Интеллектуальный планировщик"
                sx={{
                  width: "fit-content",
                  fontWeight: 700,
                  backgroundColor: "rgba(37, 99, 235, 0.10)",
                  color: "#1d4ed8",
                }}
              />
              <Typography
                variant="h2"
                sx={{
                  fontSize: { xs: "2rem", md: "3.5rem" },
                  lineHeight: 1.05,
                  fontWeight: 800,
                  color: "#0f172a",
                  maxWidth: 760,
                }}
              >
                Подбор умного дома под ваш план квартиры
              </Typography>
              <Typography
                sx={{
                  maxWidth: 700,
                  fontSize: { xs: "1rem", md: "1.15rem" },
                  color: "#475569",
                }}
              >
                Сервис работает с реальным backend API: можно создать план, следить за его
                статусом и просматривать готовые наборы устройств.
              </Typography>
            </Stack>

            <Stack direction={{ xs: "column", md: "row" }} spacing={3} alignItems="stretch">
              <Box
                sx={{
                  flex: 1.1,
                  borderRadius: 6,
                  p: { xs: 2.5, md: 3 },
                  background: "linear-gradient(180deg, #0f172a 0%, #1e293b 100%)",
                  color: "#fff",
                }}
              >
                <Stack spacing={2.5}>
                  <Typography sx={{ fontSize: "1.35rem", fontWeight: 800 }}>
                    Новый план умного дома
                  </Typography>
                  <Typography sx={{ color: "rgba(255,255,255,0.74)" }}>
                    Начните с выбора экосистемы и требований, после чего frontend отправит
                    реальный запрос на создание нового плана.
                  </Typography>
                  <Link href="/settings">
                    <Button
                      variant="contained"
                      endIcon={<ArrowForwardRoundedIcon />}
                      sx={{
                        width: "fit-content",
                        px: 2.5,
                        py: 1.2,
                        borderRadius: 3,
                        fontWeight: 800,
                        background: "linear-gradient(135deg, #22c55e 0%, #16a34a 100%)",
                        boxShadow: "0 14px 30px rgba(34,197,94,0.25)",
                      }}
                    >
                      Подобрать новый умный дом
                    </Button>
                  </Link>
                </Stack>
              </Box>

              <Stack spacing={2} sx={{ flex: 1 }}>
                <PreviewCard
                  icon={<HomeWorkRoundedIcon />}
                  title="Реальные планы из API"
                  text="На главной отображаются planning sessions из GET /api/v1/plans."
                />
                <PreviewCard
                  icon={<ChecklistRoundedIcon />}
                  title="Живые требования"
                  text="Настройки подтягивают экосистемы, пресеты и типы устройств из backend."
                />
                <PreviewCard
                  icon={<TimelineRoundedIcon />}
                  title="Статус генерации"
                  text="Страница плана опрашивает статус генерации и показывает готовые bundles."
                />
              </Stack>
            </Stack>

            <Box>
              <Typography sx={{ mb: 1.5, fontWeight: 800, color: "#0f172a" }}>
                Последние планы
              </Typography>

              {loading ? (
                <Box sx={{ py: 5, display: "grid", placeItems: "center" }}>
                  <CircularProgress />
                </Box>
              ) : error ? (
                <Alert severity="error">{error}</Alert>
              ) : sortedPlans.length === 0 ? (
                <Alert severity="info">Пока нет ни одного созданного плана.</Alert>
              ) : (
                <Stack spacing={1.5}>
                  {sortedPlans.map((plan) => (
                    <Link key={plan.plan_id} href={`/plan?id=${plan.plan_id}`}>
                      <Box
                        sx={{
                          borderRadius: 4,
                          p: 2.2,
                          backgroundColor: "rgba(255,255,255,0.78)",
                          border: "1px solid rgba(148,163,184,0.18)",
                          transition: "160ms ease",
                          "&:hover": {
                            transform: "translateY(-1px)",
                            boxShadow: "0 16px 34px rgba(15,23,42,0.10)",
                          },
                        }}
                      >
                        <Stack
                          direction={{ xs: "column", md: "row" }}
                          justifyContent="space-between"
                          spacing={1}
                        >
                          <Box>
                            <Typography sx={{ fontWeight: 800, color: "#0f172a" }}>
                              План #{plan.plan_id}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                              Создан: {new Date(plan.created_at).toLocaleString("ru-RU")}
                            </Typography>
                          </Box>
                          <Stack direction={{ xs: "column", sm: "row" }} spacing={1}>
                            <Chip label={`Бюджет: ${Math.round(plan.budget).toLocaleString("ru-RU")} ₽`} />
                            <Chip
                              label={statusLabel(plan.status)}
                              color={statusColor(plan.status)}
                              variant="outlined"
                            />
                          </Stack>
                        </Stack>
                      </Box>
                    </Link>
                  ))}
                </Stack>
              )}
            </Box>

            <Box>
              <Typography sx={{ mb: 1.5, fontWeight: 800, color: "#0f172a" }}>
                Как это работает
              </Typography>
              <Stack direction={{ xs: "column", md: "row" }} spacing={2}>
                {steps.map((step, index) => (
                  <Box
                    key={step}
                    sx={{
                      flex: 1,
                      borderRadius: 4,
                      p: 2.2,
                      backgroundColor: "rgba(255,255,255,0.72)",
                      border: "1px solid rgba(148,163,184,0.18)",
                    }}
                  >
                    <Typography sx={{ fontWeight: 800, color: "#2563eb", mb: 1 }}>
                      {`0${index + 1}`}
                    </Typography>
                    <Typography sx={{ color: "#334155" }}>{step}</Typography>
                  </Box>
                ))}
              </Stack>
            </Box>
          </Stack>
        </Box>
      </Box>
    </Box>
  );
}

function PreviewCard(props: { icon: ReactNode; title: string; text: string }) {
  return (
    <Box
      sx={{
        borderRadius: 4,
        p: 2.2,
        backgroundColor: "rgba(255,255,255,0.78)",
        border: "1px solid rgba(148,163,184,0.16)",
      }}
    >
      <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1 }}>
        <Box
          sx={{
            display: "grid",
            placeItems: "center",
            width: 42,
            height: 42,
            borderRadius: 3,
            backgroundColor: "rgba(37,99,235,0.10)",
            color: "#2563eb",
          }}
        >
          {props.icon}
        </Box>
        <Typography sx={{ fontWeight: 800, color: "#0f172a" }}>{props.title}</Typography>
      </Stack>
      <Typography sx={{ color: "#475569" }}>{props.text}</Typography>
    </Box>
  );
}

function statusLabel(status: ApiPlanSummary["status"]) {
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

function statusColor(status: ApiPlanSummary["status"]): "default" | "primary" | "success" | "error" | "warning" {
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
