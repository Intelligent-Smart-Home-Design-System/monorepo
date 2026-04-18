"use client";

import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  Divider,
  FormControlLabel,
  LinearProgress,
  Stack,
  TextField,
  Slider,
  Typography,
  alpha,
} from "@mui/material";
import ExpandMoreRoundedIcon from "@mui/icons-material/ExpandMoreRounded";
import HubRoundedIcon from "@mui/icons-material/HubRounded";
import SensorsRoundedIcon from "@mui/icons-material/SensorsRounded";
import SecurityRoundedIcon from "@mui/icons-material/SecurityRounded";
import WbIncandescentRoundedIcon from "@mui/icons-material/WbIncandescentRounded";
import ThermostatRoundedIcon from "@mui/icons-material/ThermostatRounded";
import RadarRoundedIcon from "@mui/icons-material/RadarRounded";
import type {
  Ecosystem,
  RequirementItem,
  RequirementsByTrack,
  TrackKey,
} from "../lib/types";

const ecosystemOptions: {
  value: Ecosystem;
  title: string;
  description: string;
  logo: string;
}[] = [
  {
    value: "Apple HomeKit",
    title: "Apple HomeKit",
    description: "Удобно для устройств Apple, Siri и автоматизаций внутри экосистемы Apple.",
    logo: "/brands/apple-home.svg",
  },
  {
    value: "Google Home",
    title: "Google Home",
    description: "Подходит для голосового управления через Google Assistant и Android-окружения.",
    logo: "/brands/google-home.svg",
  },
  {
    value: "Yandex Home",
    title: "Yandex Home",
    description: "Логичный выбор для Алисы, Яндекс Станции и локального русскоязычного сценария.",
    logo: "/brands/yandex-home.svg",
  },
];

const hubOptions = ["Google Home", "Яндекс Станция"];

const trackMeta: Record<
  TrackKey,
  { label: string; description: string; icon: ReactNode }
> = {
  security: {
    label: "Безопасность",
    description: "Защита входа, тревожные сценарии и контроль доступа в квартиру.",
    icon: <SecurityRoundedIcon />,
  },
  light: {
    label: "Контроль света",
    description: "Автоматизация освещения, диммирование и сценарии присутствия.",
    icon: <WbIncandescentRoundedIcon />,
  },
  climate: {
    label: "Климат-контроль",
    description: "Управление температурой, комфортом и микроклиматом по помещениям.",
    icon: <ThermostatRoundedIcon />,
  },
  perimeter: {
    label: "Охрана периметра",
    description: "Датчики дверей, окон и проникновения по внешнему контуру квартиры.",
    icon: <RadarRoundedIcon />,
  },
};

const defaultRequirements: RequirementsByTrack = {
  security: {
    score: 8,
    items: [
      {
        id: "smart-lock",
        name: "Умный замок",
        description: "Контроль доступа на входной двери.",
        count: 1,
        enabled: true,
      },
      {
        id: "motion-sensor",
        name: "Датчик движения",
        description: "Срабатывает на перемещение в коридоре или прихожей.",
        count: 2,
        enabled: true,
      },
    ],
  },
  light: {
    score: 6,
    items: [
      {
        id: "smart-bulb",
        name: "Умная лампа",
        description: "Удалённое включение, сцены и расписание освещения.",
        count: 4,
        enabled: true,
      },
      {
        id: "smart-switch",
        name: "Умный выключатель",
        description: "Управление светом без замены всех светильников.",
        count: 2,
        enabled: true,
      },
    ],
  },
  climate: {
    score: 7,
    items: [
      {
        id: "thermostat",
        name: "Термостат",
        description: "Автоматически поддерживает заданную температуру.",
        count: 1,
        enabled: true,
      },
      {
        id: "climate-sensor",
        name: "Датчик температуры и влажности",
        description: "Помогает строить точные климатические сценарии.",
        count: 3,
        enabled: true,
      },
    ],
  },
  perimeter: {
    score: 3,
    items: [
      {
        id: "contact-sensor",
        name: "Датчик открытия",
        description: "Ставит контроль на окна и двери.",
        count: 3,
        enabled: true,
      },
      {
        id: "leak-sensor",
        name: "Датчик протечки",
        description: "Фиксирует утечки воды в ванной и на кухне.",
        count: 2,
        enabled: true,
      },
    ],
  },
};

export default function SettingsPage() {
  const router = useRouter();

  const [budget, setBudget] = useState<string>("500000");
  const [ecosystem, setEcosystem] = useState<Ecosystem>("Apple HomeKit");
  const [hubs, setHubs] = useState<Record<string, boolean>>({
    "Google Home": true,
    "Яндекс Станция": false,
  });
  const [requirements, setRequirements] = useState<RequirementsByTrack>(defaultRequirements);

  const [fileName, setFileName] = useState<string>("");
  const [planDataUrl, setPlanDataUrl] = useState<string>("");
  const [planFileType, setPlanFileType] = useState<"dxf" | "png" | "">("");
  const [uploadState, setUploadState] = useState<"idle" | "uploading" | "success" | "error">("idle");
  const [progress, setProgress] = useState<number>(0);
  const [uploadError, setUploadError] = useState<string>("");

  const canStart = useMemo(() => {
    return Number(budget) > 0 && uploadState === "success" && fileName.length > 0;
  }, [budget, fileName, uploadState]);

  return (
    <Box sx={{ minHeight: "100vh", px: { xs: 2, md: 4 }, py: { xs: 3, md: 5 } }}>
      <Card sx={{ maxWidth: 820, mx: "auto", borderRadius: 6, overflow: "hidden" }}>
        <CardContent>
          <Stack spacing={3}>
            <Box>
              <Typography variant="h4" sx={{ fontWeight: 800, mb: 1 }}>
                Ввод настроек
              </Typography>
              <Typography color="text.secondary">
                Сначала выбираем экосистему, затем настраиваем треки и состав устройств, после
                чего загружаем план квартиры в формате DXF или PNG.
              </Typography>
            </Box>

            <TextField
              label="Бюджет (₽)"
              type="text"
              value={budget}
              onChange={(e) => {
                const next = e.target.value.replace(/[^\d]/g, "");
                setBudget(next);
              }}
              inputMode="numeric"
              fullWidth
            />

            <Box>
              <Typography sx={{ fontWeight: 700, mb: 1.2 }}>Выбор экосистемы</Typography>
              <Stack spacing={1.4}>
                {ecosystemOptions.map((option) => {
                  const active = option.value === ecosystem;

                  return (
                    <Box
                      key={option.value}
                      onClick={() => setEcosystem(option.value)}
                      sx={{
                        cursor: "pointer",
                        borderRadius: 4,
                        p: 2,
                        border: active ? "2px solid #2563eb" : "1px solid rgba(148,163,184,0.32)",
                        background: active
                          ? "linear-gradient(135deg, rgba(37,99,235,0.12), rgba(59,130,246,0.04))"
                          : "#fff",
                        transition: "160ms ease",
                        "&:hover": {
                          borderColor: "#60a5fa",
                          transform: "translateY(-1px)",
                        },
                      }}
                    >
                      <Stack direction="row" spacing={2} alignItems="center">
                        <Box
                          sx={{
                            width: 52,
                            height: 52,
                            borderRadius: 3,
                            display: "grid",
                            placeItems: "center",
                            backgroundColor: active ? "rgba(37,99,235,0.10)" : "rgba(15,23,42,0.05)",
                          }}
                        >
                          <img
                            src={option.logo}
                            alt={`${option.title} logo`}
                            style={{ width: 34, height: 34, objectFit: "contain", display: "block" }}
                          />
                        </Box>
                        <Box sx={{ flex: 1 }}>
                          <Stack
                            direction={{ xs: "column", sm: "row" }}
                            spacing={1}
                            alignItems={{ xs: "flex-start", sm: "center" }}
                            sx={{ mb: 0.5 }}
                          >
                            <Typography sx={{ fontWeight: 800 }}>{option.title}</Typography>
                            {active && <Chip size="small" label="Выбрано" color="primary" />}
                          </Stack>
                          <Typography variant="body2" color="text.secondary">
                            {option.description}
                          </Typography>
                        </Box>
                      </Stack>
                    </Box>
                  );
                })}
              </Stack>
            </Box>

            <Box>
              <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                <HubRoundedIcon sx={{ color: "#2563eb" }} />
                <Typography sx={{ fontWeight: 700 }}>Дополнительные хабы</Typography>
              </Stack>
              <Stack>
                {hubOptions.map((hub) => (
                  <FormControlLabel
                    key={hub}
                    control={
                      <Checkbox
                        checked={hubs[hub]}
                        onChange={(e) => setHubs((prev) => ({ ...prev, [hub]: e.target.checked }))}
                      />
                    }
                    label={hub}
                  />
                ))}
              </Stack>
            </Box>

            <Box>
              <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                <SensorsRoundedIcon sx={{ color: "#2563eb" }} />
                <Typography sx={{ fontWeight: 700 }}>Оценка значимости треков и состав устройств</Typography>
              </Stack>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1.8 }}>
                Каждый трек можно раскрыть: внутри видно, какие типы устройств он предполагает,
                сколько их планируется и нужно ли вообще брать конкретную позицию.
              </Typography>

              <Stack spacing={1.2}>
                {(Object.keys(trackMeta) as TrackKey[]).map((trackKey) => (
                  <Accordion
                    key={trackKey}
                    disableGutters
                    sx={{
                      borderRadius: 4,
                      overflow: "hidden",
                      border: "1px solid rgba(148,163,184,0.26)",
                      "&:before": { display: "none" },
                    }}
                  >
                    <AccordionSummary expandIcon={<ExpandMoreRoundedIcon />}>
                      <Stack spacing={1} sx={{ width: "100%" }}>
                        <Stack
                          direction={{ xs: "column", md: "row" }}
                          justifyContent="space-between"
                          spacing={1.2}
                        >
                          <Stack direction="row" spacing={1.2} alignItems="center">
                            <Box
                              sx={{
                                display: "grid",
                                placeItems: "center",
                                width: 38,
                                height: 38,
                                borderRadius: 3,
                                backgroundColor: "rgba(37,99,235,0.10)",
                                color: "#2563eb",
                              }}
                            >
                              {trackMeta[trackKey].icon}
                            </Box>
                            <Box>
                              <Typography sx={{ fontWeight: 800 }}>{trackMeta[trackKey].label}</Typography>
                              <Typography variant="body2" color="text.secondary">
                                {trackMeta[trackKey].description}
                              </Typography>
                            </Box>
                          </Stack>
                          <Chip
                            label={`Уровень важности: ${requirements[trackKey].score}/10`}
                            sx={{ alignSelf: "flex-start" }}
                          />
                        </Stack>
                      </Stack>
                    </AccordionSummary>
                    <AccordionDetails sx={{ pt: 0 }}>
                      <TrackScoreRow
                        label={trackMeta[trackKey].label}
                        value={requirements[trackKey].score}
                        onChange={(value) =>
                          setRequirements((prev) => ({
                            ...prev,
                            [trackKey]: { ...prev[trackKey], score: value },
                          }))
                        }
                      />

                      <Divider sx={{ my: 2 }} />

                      <Stack spacing={1.2}>
                        {requirements[trackKey].items.map((item) => (
                          <RequirementRow
                            key={item.id}
                            item={item}
                            onChange={(nextItem) => {
                              setRequirements((prev) => ({
                                ...prev,
                                [trackKey]: {
                                  ...prev[trackKey],
                                  items: prev[trackKey].items.map((current) =>
                                    current.id === item.id ? nextItem : current
                                  ),
                                },
                              }));
                            }}
                          />
                        ))}
                      </Stack>
                    </AccordionDetails>
                  </Accordion>
                ))}
              </Stack>
            </Box>

            <Box>
              <Typography sx={{ fontWeight: 700, mb: 1 }}>План квартиры</Typography>

              <Button variant="outlined" component="label">
                Загрузить файл DXF/PNG
                <input
                  hidden
                  type="file"
                  accept=".dxf,.png,image/png"
                  onChange={(e) => {
                    const f = e.target.files?.[0];
                    if (!f) return;

                    setFileName(f.name);
                    setUploadError("");

                    if (f.size > 10 * 1024 * 1024) {
                      setUploadState("error");
                      setProgress(0);
                      setPlanDataUrl("");
                      setPlanFileType("");
                      setUploadError("Файл слишком большой. Максимальный размер: 10 МБ.");
                      return;
                    }

                    const lowerName = f.name.toLowerCase();
                    const isDxf = lowerName.endsWith(".dxf");
                    const isPng = lowerName.endsWith(".png") || f.type === "image/png";
                    const isSupported = isDxf || isPng;

                    if (!isSupported) {
                      setUploadState("error");
                      setProgress(0);
                      setPlanDataUrl("");
                      setPlanFileType("");
                      setUploadError("Сейчас поддерживаем только файлы формата DXF или PNG.");
                      return;
                    }

                    setUploadState("uploading");
                    setProgress(30);

                    if (isPng) {
                      const reader = new FileReader();
                      reader.onload = () => {
                        setPlanDataUrl(String(reader.result));
                        setPlanFileType("png");
                        setProgress(100);
                        setUploadState("success");
                      };
                      reader.onerror = () => {
                        setUploadState("error");
                        setProgress(0);
                        setPlanDataUrl("");
                        setPlanFileType("");
                        setUploadError("Не удалось прочитать PNG-файл.");
                      };
                      reader.readAsDataURL(f);
                      return;
                    }

                    window.setTimeout(() => {
                      setPlanDataUrl("");
                      setPlanFileType("dxf");
                      setProgress(100);
                      setUploadState("success");
                    }, 350);
                  }}
                />
              </Button>

              <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                Сейчас поддерживаем `.dxf` и `.png`. PNG будет показан на странице плана, DXF пока
                сохраняется как загруженный файл без визуального превью.
              </Typography>

              {uploadState === "uploading" && (
                <Box sx={{ mt: 1.2 }}>
                  <LinearProgress variant="determinate" value={progress} />
                  <Typography sx={{ mt: 0.5 }} variant="body2" color="text.secondary">
                    Загрузка: {progress}%
                  </Typography>
                </Box>
              )}

              {uploadState === "success" && (
                <Alert sx={{ mt: 1.2 }} severity="success">
                  Загружено успешно: {fileName}
                </Alert>
              )}

              {uploadState === "success" && planFileType === "png" && planDataUrl && (
                <Box sx={{ mt: 1.5, borderRadius: 3, overflow: "hidden", border: "1px solid rgba(148,163,184,0.25)" }}>
                  <img
                    src={planDataUrl}
                    alt="Превью плана квартиры"
                    style={{ width: "100%", maxHeight: 260, objectFit: "contain", display: "block" }}
                  />
                </Box>
              )}

              {uploadState === "error" && (
                <Alert sx={{ mt: 1.2 }} severity="error">
                  Ошибка загрузки: {uploadError}
                </Alert>
              )}
            </Box>

            <Button
              variant="contained"
              fullWidth
              size="large"
              disabled={!canStart}
              onClick={() => {
                const tracks = (Object.keys(requirements) as TrackKey[]).reduce(
                  (acc, key) => ({ ...acc, [key]: requirements[key].score }),
                  {} as Record<TrackKey, number>
                );
                const data = {
                  budget: Number(budget),
                  ecosystem,
                  hubs,
                  tracks,
                  requirements,
                  fileName,
                  planDataUrl,
                  planFileType,
                };
                localStorage.setItem("settings", JSON.stringify(data));
                router.push("/plan");
              }}
            >
              Запустить подбор
            </Button>

            <Typography variant="body2" color="text.secondary">
              (Кнопка активируется после успешной загрузки плана)
            </Typography>
          </Stack>
        </CardContent>
      </Card>
    </Box>
  );
}

function TrackScoreRow(props: { label: string; value: number; onChange: (v: number) => void }) {
  return (
    <Box>
      <Typography variant="body2" sx={{ mb: 0.5 }}>
        {props.label}: {props.value}
      </Typography>
      <Slider
        value={props.value}
        min={0}
        max={10}
        step={1}
        onChange={(_, v) => props.onChange(v as number)}
      />
    </Box>
  );
}

function RequirementRow(props: {
  item: RequirementItem;
  onChange: (next: RequirementItem) => void;
}) {
  return (
    <Box
      sx={(theme) => ({
        borderRadius: 3,
        p: 1.5,
        border: "1px solid rgba(148,163,184,0.24)",
        backgroundColor: props.item.enabled
          ? alpha(theme.palette.primary.light, 0.08)
          : "rgba(148,163,184,0.08)",
      })}
    >
      <Stack
        direction={{ xs: "column", md: "row" }}
        spacing={1.5}
        justifyContent="space-between"
        alignItems={{ xs: "flex-start", md: "center" }}
      >
        <Box sx={{ flex: 1 }}>
          <FormControlLabel
            control={
              <Checkbox
                checked={props.item.enabled}
                onChange={(event) =>
                  props.onChange({ ...props.item, enabled: event.target.checked })
                }
              />
            }
            label={<Typography sx={{ fontWeight: 700 }}>{props.item.name}</Typography>}
            sx={{ alignItems: "flex-start", m: 0 }}
          />
          <Typography variant="body2" color="text.secondary" sx={{ ml: 4.5, mt: -0.4 }}>
            {props.item.description}
          </Typography>
        </Box>

        <TextField
          label="Количество"
          type="number"
          size="small"
          value={props.item.count}
          onChange={(event) =>
            props.onChange({
              ...props.item,
              count: Math.max(0, Number(event.target.value) || 0),
            })
          }
          inputProps={{ min: 0 }}
          sx={{ width: { xs: "100%", md: 130 } }}
        />
      </Stack>
    </Box>
  );
}
