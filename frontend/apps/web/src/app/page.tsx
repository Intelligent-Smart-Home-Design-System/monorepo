import type { ReactNode } from "react";
import ArrowForwardRoundedIcon from "@mui/icons-material/ArrowForwardRounded";
import AutoAwesomeRoundedIcon from "@mui/icons-material/AutoAwesomeRounded";
import ChecklistRoundedIcon from "@mui/icons-material/ChecklistRounded";
import HomeWorkRoundedIcon from "@mui/icons-material/HomeWorkRounded";
import TimelineRoundedIcon from "@mui/icons-material/TimelineRounded";
import { Box, Button, Chip, Stack, Typography } from "@mui/material";
import Link from "next/link";

const steps = [
  "Загрузите план квартиры в формате DXF.",
  "Выберите экосистему и укажите приоритеты по трекам.",
  "Настройте состав устройств по каждому уровню и запустите подбор.",
];

export default function Home() {
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
                Сервис помогает собрать набор устройств под бюджет, экосистему и реальные
                требования по безопасности, свету, климату и периметру.
              </Typography>
            </Stack>

            <Stack direction={{ xs: "column", md: "row" }} spacing={3} alignItems="stretch">
              <Box
                sx={{
                  flex: 1.2,
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
                    Начните с ввода параметров, загрузки DXF-плана и настройки уровней устройств.
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
                  title="DXF-план квартиры"
                  text="Поддерживаем загрузку инженерного плана для дальнейшего разбора."
                />
                <PreviewCard
                  icon={<ChecklistRoundedIcon />}
                  title="Гибкая настройка требований"
                  text="Для каждого трека можно раскрыть состав устройств, поменять количество и отключить лишнее."
                />
                <PreviewCard
                  icon={<TimelineRoundedIcon />}
                  title="Прозрачный сценарий подбора"
                  text="Сначала приветственная страница, затем настройки, а детализацию плана вынесем отдельным этапом."
                />
              </Stack>
            </Stack>

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
