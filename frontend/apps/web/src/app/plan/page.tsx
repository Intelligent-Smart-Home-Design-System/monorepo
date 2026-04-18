"use client";

import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import LockRoundedIcon from "@mui/icons-material/LockRounded";
import SensorsRoundedIcon from "@mui/icons-material/SensorsRounded";
import ThermostatRoundedIcon from "@mui/icons-material/ThermostatRounded";
import WbIncandescentRoundedIcon from "@mui/icons-material/WbIncandescentRounded";
import Image from "next/image";
import { useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import type { Ecosystem, TrackKey } from "../lib/types";
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  Stack,
  Typography,
  Switch,
  FormControlLabel,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Avatar,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  alpha,
} from "@mui/material";

type Device = {
  id: string;
  name: string;
  type: string;
  ecosystem: Ecosystem;
  x: number;
  y: number;
  price: number;
  impacts: Partial<Record<TrackKey, number>>;
};

type Settings = {
  planDataUrl?: string;
  planFileType?: "dxf" | "png" | "";
  budget: number;
  ecosystem: Ecosystem;
  hubs: Record<string, boolean>;
  tracks: Record<TrackKey, number>;
  fileName?: string;
};

type MarketplaceOffer = {
  name: "Яндекс Маркет" | "Ozon" | "Wildberries";
  price: number;
  url?: string;
};

type AnalogItem = {
  name: string;
  price: number;
  img: string;
  url?: string;
};

type DeviceCardInfo = {
  title: string;         // Xiaomi Mijia
  description: string;   // текст под названием
  img: string;           // /devices/lock.png
  offers: MarketplaceOffer[];
  analogs: AnalogItem[];
};

const defaultDevices: Device[] = [
  {
    id: "lock",
    name: "Умный замок",
    type: "Smart Lock",
    ecosystem: "Apple HomeKit",
    x: 62,
    y: 47,
    price: 10600,
    impacts: { security: 9 },
  },
  {
    id: "light",
    name: "Свет в гостиной",
    type: "Light",
    ecosystem: "Yandex Home",
    x: 78,
    y: 68,
    price: 2500,
    impacts: { light: 8 },
  },
  {
    id: "climate",
    name: "Климат-контроль",
    type: "Thermostat",
    ecosystem: "Google Home",
    x: 35,
    y: 35,
    price: 8900,
    impacts: { climate: 9 },
  },
  {
    id: "perimeter",
    name: "Датчик открытия двери/окна",
    type: "Contact Sensor",
    ecosystem: "Yandex Home",
    x: 20,
    y: 60,
    price: 1700,
    impacts: { perimeter: 7, security: 3 },
  },
];


export default function PlanPage() {
  const router = useRouter();

  const [devices, setDevices] = useState<Device[]>(() => {
    if (typeof window === "undefined") return defaultDevices;

    try {
      const raw = localStorage.getItem("devices");
      if (raw) {
        const saved = JSON.parse(raw);
        if (Array.isArray(saved) && saved.length > 0) return saved as Device[];
      }
    } catch { }

    return defaultDevices;
  });


  useEffect(() => {
    localStorage.setItem("devices", JSON.stringify(devices));
  }, [devices]);


  useEffect(() => {
    localStorage.setItem("devices", JSON.stringify(devices));
  }, [devices]);


  const [settings, setSettings] = useState<Settings | null>(null);
  const [settingsReady, setSettingsReady] = useState(false);
  const [selectedId, setSelectedId] = useState<string>("lock");
  const [onlyRecommended, setOnlyRecommended] = useState(false);
  const [addOpen, setAddOpen] = useState(false);

  const [newDeviceId, setNewDeviceId] = useState("lock");
  const [newDeviceName, setNewDeviceName] = useState("Новое устройство");
  const [newDeviceEco, setNewDeviceEco] = useState<Ecosystem>("Apple HomeKit");
  const [newDevicePrice, setNewDevicePrice] = useState(3000);

  const planRef = useRef<HTMLDivElement | null>(null);
  const [dragId, setDragId] = useState<string | null>(null);

  useEffect(() => {
    const raw = localStorage.getItem("settings");
    if (!raw) {
      router.push("/settings");
      return;
    }
    try {
      setSettings(JSON.parse(raw));
      setSettingsReady(true);
    } catch {
      router.push("/settings");
    }
  }, [router]);

  useEffect(() => {
    const rawDevices = localStorage.getItem("devices");
    if (rawDevices) {
      try {
        const saved = JSON.parse(rawDevices) as Device[];
        if (Array.isArray(saved) && saved.length > 0) {
          setDevices(saved);
          return;
        }
      } catch { }
    }

    const rawPos = localStorage.getItem("devicePositions");
    if (!rawPos) return;

    try {
      const pos = JSON.parse(rawPos) as Record<string, { x: number; y: number }>;
      setDevices((prev) =>
        prev.map((d) => (pos[d.id] ? { ...d, x: pos[d.id].x, y: pos[d.id].y } : d))
      );
    } catch { }
  }, []);

  const scored = useMemo(() => {
    if (!settings) return [];

    const filtered = devices.filter((d) => d.ecosystem === settings.ecosystem);

    const scoreOf = (d: Device) => {
      let s = 0;
      for (const k of Object.keys(settings.tracks) as TrackKey[]) {
        const w = settings.tracks[k] ?? 0;
        const impact = d.impacts[k] ?? 0;
        s += w * impact;
      }
      return s;
    };

    return filtered
      .map((d) => ({
        device: d,
        score: scoreOf(d),
        valuePerRub: scoreOf(d) / Math.max(1, d.price),
      }))
      .sort((a, b) => b.valuePerRub - a.valuePerRub);
  }, [devices, settings]);

  const recommended = useMemo(() => {
    if (!settings) return [];
    let sum = 0;
    const picked: typeof scored = [];

    for (const item of scored) {
      if (sum + item.device.price <= settings.budget) {
        picked.push(item);
        sum += item.device.price;
      }
    }
    return picked;
  }, [scored, settings]);

  const recommendedIds = useMemo(() => {
    return new Set(recommended.map((x) => x.device.id));
  }, [recommended]);

  const visibleDevices = useMemo(() => {
    if (!settings) return devices;
    const ecosystemFiltered = devices.filter((d) => d.ecosystem === settings.ecosystem);
    if (!onlyRecommended) return ecosystemFiltered;
    return ecosystemFiltered.filter((d) => recommendedIds.has(d.id));
  }, [devices, onlyRecommended, recommendedIds, settings]);

  const selected = visibleDevices.find((d) => d.id === selectedId) ?? visibleDevices[0];

  const total = useMemo(() => {
    return recommended.reduce((acc, x) => acc + x.device.price, 0);
  }, [recommended]);

  const left = settings ? settings.budget - total : 0;

  const savePositions = (next: Device[]) => {
    const pos: Record<string, { x: number; y: number }> = {};
    for (const d of next) pos[d.id] = { x: d.x, y: d.y };
    localStorage.setItem("devicePositions", JSON.stringify(pos));
  };

  const openUrl = (url?: string) => {
    if (!url) return;
    window.open(url, "_blank", "noopener,noreferrer");
  };

  const impactsByKind = (kind: string): Partial<Record<TrackKey, number>> => {
    switch (kind) {
      case "lock":
        return { security: 9 };
      case "light":
        return { light: 8 };
      case "climate":
        return { climate: 9 };
      case "perimeter":
        return { perimeter: 7, security: 3 };
      default:
        return {};
    }
  };

  const iconByDevice = (device: Device) => {
    const kind = device.id.split("-")[0];
    const type = device.type.toLowerCase();

    if (kind === "light" || type.includes("light")) return <WbIncandescentRoundedIcon fontSize="inherit" />;
    if (kind === "climate" || type.includes("thermostat")) return <ThermostatRoundedIcon fontSize="inherit" />;
    if (kind === "perimeter" || type.includes("sensor")) return <SensorsRoundedIcon fontSize="inherit" />;
    return <LockRoundedIcon fontSize="inherit" />;
  };


  const addDevice = () => {
    const uniqueId = `${newDeviceId}-${Date.now()}`;

    const nextDevice: Device = {
      id: uniqueId,
      name: newDeviceName.trim() || "Новое устройство",
      type: newDeviceId,
      ecosystem: newDeviceEco,
      x: 50,
      y: 50,
      price: Number(newDevicePrice) || 0,
      impacts: impactsByKind(newDeviceId),
    };

    setDevices((prev) => {
      const next = [...prev, nextDevice];
      savePositions(next);
      return next;
    });

    setSelectedId(uniqueId);
    setAddOpen(false);
  };



  const cardInfoById: Record<string, DeviceCardInfo> = {
    lock: {
      title: "Xiaomi Mijia",
      description:
        "Способы разблокировки: отпечаток пальца, пароль, временный пароль, Bluetooth, приложение, аварийный ключ.",
      img: "/devices/lock.png",
      offers: [
        { name: "Яндекс Маркет", price: 19950, url: "https://market.yandex.ru/" },
        { name: "Ozon", price: 20450, url: "https://www.ozon.ru/" },
        { name: "Wildberries", price: 21300, url: "https://www.wildberries.ru/" },
      ],
      analogs: [
        {
          name: "Aqara Door lock N100",
          price: 18990,
          img: "/devices/lock.png",
          url: "https://market.yandex.ru/",
        },
        {
          name: "Samsung Smart Lock",
          price: 25990,
          img: "/devices/lock.png",
          url: "https://www.ozon.ru/",
        },
      ],
    },

    light: {
      title: "Умный свет",
      description: "Лампы и выключатели с управлением через приложение и голос.",
      img: "/devices/light.png",
      offers: [
        { name: "Яндекс Маркет", price: 2490, url: "https://market.yandex.ru/" },
        { name: "Ozon", price: 2690, url: "https://www.ozon.ru/" },
        { name: "Wildberries", price: 2390, url: "https://www.wildberries.ru/" },
      ],
      analogs: [
        {
          name: "Yeelight Bulb",
          price: 2190,
          img: "/devices/light.png",
          url: "https://market.yandex.ru/",
        },
        {
          name: "Philips Hue",
          price: 3990,
          img: "/devices/light.png",
          url: "https://www.ozon.ru/",
        },
      ],
    },

    climate: {
      title: "Thermostat Pro",
      description: "Поддерживает авто-режим, расписания и управление удалённо.",
      img: "/devices/thermostat.png",
      offers: [
        { name: "Яндекс Маркет", price: 8990, url: "https://market.yandex.ru/" },
        { name: "Ozon", price: 9200, url: "https://www.ozon.ru/" },
        { name: "Wildberries", price: 8700, url: "https://www.wildberries.ru/" },
      ],

      analogs: [
        {
          name: "Nest Thermostat",
          price: 12990,
          img: "/devices/thermostat.png",
          url: "https://market.yandex.ru/",
        },
        {
          name: "Tado Smart",
          price: 11990,
          img: "/devices/thermostat.png",
          url: "https://www.ozon.ru/",
        },
      ],
    },

    perimeter: {
      title: "Door/Window Sensor",
      description:
        "Срабатывает при открытии, отправляет уведомления, работает от батарейки.",
      img: "/devices/sensor.png",
      offers: [
        { name: "Яндекс Маркет", price: 1690, url: "https://market.yandex.ru/" },
        { name: "Ozon", price: 1750, url: "https://www.ozon.ru/" },
        { name: "Wildberries", price: 1590, url: "https://www.wildberries.ru/" },
      ],
      analogs: [
        {
          name: "Aqara Sensor",
          price: 1990,
          img: "/devices/sensor.png",
          url: "https://market.yandex.ru/",
        },
        {
          name: "Sonoff Sensor",
          price: 990,
          img: "/devices/sensor.png",
          url: "https://www.ozon.ru/",
        },
      ],
    },
  };

  const selectedInfo = selected
    ? cardInfoById[selected.id] ??
    cardInfoById[String(selected.id).split("-")[0]]
    : null;

  const bestOffer = useMemo(() => {
    if (!selectedInfo?.offers?.length) return null;
    return selectedInfo.offers.reduce(
      (min, o) => (o.price < min.price ? o : min),
      selectedInfo.offers[0]
    );
  }, [selectedInfo]);


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
              <Typography
                variant="h4"
                sx={{ fontWeight: 900, letterSpacing: "-0.04em", mb: 0.8 }}
              >
                План квартиры и устройства
              </Typography>
              <Typography sx={{ color: "rgba(255,255,255,0.72)", maxWidth: 620 }}>
                Перетаскивайте устройства по плану, сравнивайте предложения и контролируйте
                итоговую стоимость набора.
              </Typography>
            </Box>

            <Stack direction={{ xs: "column", sm: "row" }} spacing={1} sx={{ width: { xs: "100%", lg: "auto" } }}>
            <Button
              variant="contained"
              onClick={() => setAddOpen(true)}
              sx={{
                fontWeight: 900,
                borderRadius: 3,
                background: "linear-gradient(135deg, #2563eb, #38bdf8)",
                boxShadow: "0 14px 28px rgba(37,99,235,0.32)",
              }}
            >
              + Добавить устройство
            </Button>

            <Button
              variant="contained"
              onClick={() => router.push("/analytics")}
              sx={{
                fontWeight: 800,
                borderRadius: 3,

                background: "linear-gradient(135deg, #6366f1, #3b82f6)",
                color: "#fff",

                boxShadow: "0 4px 12px rgba(99,102,241,0.35)",

                "&:hover": {
                  background: "linear-gradient(135deg, #4f46e5, #2563eb)",
                  boxShadow: "0 6px 16px rgba(99,102,241,0.45)",
                },
              }}
            >
              Аналитика
            </Button>

            <Button
              variant="contained"
              onClick={() => router.push("/scenes")}
              sx={{
                fontWeight: 800,
                borderRadius: 3,
                background: "linear-gradient(135deg, #10b981, #22c55e)",
                color: "#fff",
                boxShadow: "0 4px 12px rgba(16,185,129,0.35)",
                "&:hover": {
                  background: "linear-gradient(135deg, #059669, #16a34a)",
                  boxShadow: "0 6px 16px rgba(16,185,129,0.45)",
                },
              }}
            >
              Сценарии
            </Button>


            <Button
              variant="contained"
              onClick={() => {
                localStorage.removeItem("devices");
                localStorage.removeItem("devicePositions");
                router.push("/settings");
              }}
              sx={{
                background: "#fff",
                color: "#1f2937",
                borderRadius: 3,
                fontWeight: 800,
                boxShadow: "0 2px 6px rgba(0,0,0,0.08)",
                "&:hover": {
                  background: "#f9fafb",
                  boxShadow: "0 4px 12px rgba(0,0,0,0.12)",
                },
              }}
            >
              Назад к вводу настроек
            </Button>

          </Stack>
          </Stack>
        </Box>

        {/* Итоги */}
        <Card
          sx={{
            borderRadius: 5,
            mb: 2.5,
            background: "rgba(255,255,255,0.94)",
            boxShadow: "0 22px 60px rgba(15,23,42,0.18)",
            border: "1px solid rgba(226,232,240,0.8)",
          }}
        >
          <CardContent>
            <Stack
              direction={{ xs: "column", md: "row" }}
              justifyContent="space-between"
              alignItems={{ xs: "flex-start", md: "center" }}
              spacing={1}
            >
              <Stack spacing={0.3}>
                <Typography sx={{ fontWeight: 900, color: "#0f172a" }}>Итого по подбору</Typography>
                <Typography color="text.secondary">
                  Экосистема: <b>{settings?.ecosystem ?? "—"}</b>
                </Typography>
              </Stack>

              <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                <StatPill label="Бюджет" value={`${(settings?.budget ?? 0).toLocaleString("ru-RU")} ₽`} />
                <StatPill label="Набор" value={`${total.toLocaleString("ru-RU")} ₽`} />
                <StatPill
                  label="Остаток"
                  value={`${left.toLocaleString("ru-RU")} ₽`}
                  tone={left < 0 ? "danger" : "success"}
                />
              </Stack>

              <FormControlLabel
                control={
                  <Switch
                    checked={onlyRecommended}
                    onChange={(e) => setOnlyRecommended(e.target.checked)}
                  />
                }
                label="Только рекомендованные"
              />
            </Stack>

            <Divider sx={{ my: 1.5 }} />

            <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap" }}>
              {["Apple HomeKit", "Google Home", "Yandex Home"].map((label) => (
                <Chip
                  key={label}
                  label={label}
                  sx={{
                    fontWeight: 700,
                    backgroundColor: label === settings?.ecosystem ? "#dbeafe" : "#f1f5f9",
                    color: label === settings?.ecosystem ? "#1d4ed8" : "#475569",
                  }}
                />
              ))}
            </Stack>
          </CardContent>
        </Card>

        <Stack direction={{ xs: "column", md: "row" }} spacing={2.5}>
          {/* План */}
          <Card
            sx={{
              flex: 1,
              borderRadius: 5,
              background: "rgba(255,255,255,0.96)",
              boxShadow: "0 26px 70px rgba(15,23,42,0.22)",
              border: "1px solid rgba(226,232,240,0.8)",
            }}
          >
            <CardContent>
              <Stack
                direction={{ xs: "column", sm: "row" }}
                justifyContent="space-between"
                alignItems={{ xs: "flex-start", sm: "center" }}
                spacing={1}
                sx={{ mb: 1.5 }}
              >
                <Box>
                  <Typography sx={{ fontWeight: 900, color: "#0f172a" }}>
                    Интерактивный план
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Кликните на устройство или перетащите пин в нужную комнату.
                  </Typography>
                </Box>
                <Chip label={`${visibleDevices.length} устройств`} sx={{ fontWeight: 800 }} />
              </Stack>

              <Box
                ref={planRef}
                onPointerMove={(e) => {
                  if (!dragId) return;
                  const el = planRef.current;
                  if (!el) return;

                  const r = el.getBoundingClientRect();
                  const px = ((e.clientX - r.left) / r.width) * 100;
                  const py = ((e.clientY - r.top) / r.height) * 100;

                  const x = Math.max(0, Math.min(100, px));
                  const y = Math.max(0, Math.min(100, py));

                  setDevices((prev) => {
                    const next = prev.map((d) => (d.id === dragId ? { ...d, x, y } : d));
                    savePositions(next);
                    return next;
                  });
                }}
                onPointerUp={() => setDragId(null)}
                onPointerLeave={() => setDragId(null)}
                sx={{
                  position: "relative",
                  width: "100%",
                  aspectRatio: "16 / 10",
                  borderRadius: 4,
                  overflow: "hidden",
                  background:
                    "linear-gradient(145deg, #e2e8f0 0%, #f8fafc 45%, #dbeafe 100%)",
                  touchAction: "none",
                  border: "1px solid rgba(148,163,184,0.34)",
                  boxShadow: "inset 0 1px 0 rgba(255,255,255,0.85)",
                }}
              >

                {!settingsReady ? (
                  <Box
                    sx={{
                      position: "absolute",
                      inset: 0,
                      display: "grid",
                      placeItems: "center",
                      background: "#e9edf3",
                    }}
                  >
                    <Typography variant="body2" color="text.secondary">
                      Загружаем план...
                    </Typography>
                  </Box>
                ) : settings?.planDataUrl ? (
                  <Image
                    src={settings.planDataUrl}
                    alt="Floor plan"
                    fill
                    style={{ objectFit: "contain" }}
                    priority
                    unoptimized
                  />
                ) : settings?.planFileType === "dxf" ? (
                  <Box
                    sx={{
                      position: "absolute",
                      inset: 0,
                      display: "grid",
                      placeItems: "center",
                      px: 3,
                      textAlign: "center",
                      background:
                        "linear-gradient(135deg, rgba(37,99,235,0.08), rgba(15,23,42,0.04))",
                    }}
                  >
                    <Stack spacing={1}>
                      <Typography sx={{ fontWeight: 800, color: "#1e293b" }}>
                        DXF-файл загружен
                      </Typography>
                      <Typography color="text.secondary">
                        {settings.fileName}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Превью DXF пока не отрисовывается в браузере. Для визуального плана
                        загрузите PNG.
                      </Typography>
                    </Stack>
                  </Box>
                ) : (
                  <Image
                    src="/floorplan.png"
                    alt="Floor plan"
                    fill
                    style={{ objectFit: "cover" }}
                    priority
                    unoptimized
                  />
                )}
                {visibleDevices.map((d) => {
                  const isRec = recommendedIds.has(d.id);
                  const isSelected = d.id === (selected?.id ?? "");
                  const markerColor =
                    d.ecosystem === "Apple HomeKit"
                      ? "#f59e0b"
                      : d.ecosystem === "Google Home"
                        ? "#22c55e"
                        : "#3b82f6";

                  return (
                    <Box
                      key={d.id}
                      onPointerDown={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        setSelectedId(d.id);
                        setDragId(d.id);
                      }}
                      onClick={() => setSelectedId(d.id)}
                      sx={{
                        position: "absolute",
                        left: `${d.x}%`,
                        top: `${d.y}%`,
                        transform: "translate(-50%, -72%)",
                        width: isRec ? 42 : 38,
                        height: isRec ? 42 : 38,
                        borderRadius: "16px 16px 16px 4px",
                        cursor: "grab",
                        display: "grid",
                        placeItems: "center",
                        fontSize: isRec ? 24 : 22,
                        color: markerColor,
                        border: "3px solid rgba(255,255,255,0.96)",
                        boxShadow: isSelected
                          ? `0 16px 32px rgba(15,23,42,0.34), 0 0 0 5px ${markerColor}42, inset 0 2px 0 rgba(255,255,255,0.95)`
                          : "0 14px 28px rgba(15,23,42,0.28), inset 0 2px 0 rgba(255,255,255,0.95)",
                        background:
                          "linear-gradient(145deg, #ffffff 0%, #eef2f7 46%, #dbe3ef 100%)",
                        outline: "none",
                        opacity: onlyRecommended && !isRec ? 0.25 : 1,
                        userSelect: "none",
                        transition: "transform 140ms ease, box-shadow 140ms ease",
                        "&:hover": {
                          transform: "translate(-50%, -76%) scale(1.08)",
                          boxShadow: `0 18px 36px rgba(15,23,42,0.36), 0 0 0 5px ${markerColor}36, inset 0 2px 0 rgba(255,255,255,0.95)`,
                        },
                        "&::after": {
                          content: '""',
                          position: "absolute",
                          left: 3,
                          bottom: 3,
                          width: 10,
                          height: 10,
                          borderRadius: "0 0 0 3px",
                          background: markerColor,
                          boxShadow: "inset 0 1px 0 rgba(255,255,255,0.35)",
                        },
                        "&::before": {
                          content: '""',
                          position: "absolute",
                          inset: 4,
                          borderRadius: "13px 13px 13px 3px",
                          border: `2px solid ${markerColor}`,
                          opacity: 0.38,
                        },
                      }}
                      title={isRec ? `${d.name} (рекомендуется)` : d.name}
                    >
                      {iconByDevice(d)}
                    </Box>
                  );
                })}
              </Box>
            </CardContent>
          </Card>

          {/* Карточка */}
          <Card
            sx={{
              width: { xs: "100%", md: 420 },
              borderRadius: 5,
              background: "rgba(255,255,255,0.96)",
              boxShadow: "0 26px 70px rgba(15,23,42,0.22)",
              border: "1px solid rgba(226,232,240,0.8)",
            }}
          >
            <CardContent>
              <Typography sx={{ fontWeight: 900, mb: 1, color: "#0f172a" }}>Карточка устройства</Typography>
              <Divider sx={{ mb: 2 }} />

              {!selected || !selectedInfo ? (
                <Typography color="text.secondary">Выбери устройство на плане</Typography>
              ) : (
                <Stack spacing={2}>
                  {/* Верх: фото + текст */}
                  <Stack direction="row" spacing={2} alignItems="center">
                    <Box
                      sx={{
                        width: 96,
                        height: 96,
                        borderRadius: 4,
                        overflow: "hidden",
                        background:
                          "radial-gradient(circle at 35% 28%, #ffffff, #e2e8f0 72%)",
                        boxShadow: "inset 0 1px 0 rgba(255,255,255,0.9), 0 12px 26px rgba(15,23,42,0.12)",
                        flexShrink: 0,
                      }}
                    >
                      <img
                        src={selectedInfo.img}
                        alt={selectedInfo.title}
                        style={{ width: "100%", height: "100%", objectFit: "cover", display: "block" }}
                      />
                    </Box>

                    <Box>
                      <Typography variant="h6" sx={{ fontWeight: 800, lineHeight: 1.1 }}>
                        {selected.name}
                      </Typography>
                      <Typography sx={{ fontWeight: 700, mt: 0.2 }}>{selectedInfo.title}</Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ mt: 0.6 }}>
                        {selectedInfo.description}
                      </Typography>
                    </Box>
                  </Stack>

                  {/* Маркетплейсы */}
                  <Stack spacing={1.2}>
                    {selectedInfo.offers.map((o) => (
                      <Stack
                        key={o.name}
                        direction="row"
                        alignItems="center"
                        justifyContent="space-between"
                        onClick={() => openUrl(o.url)}
                        sx={{
                          p: 1.3,
                          borderRadius: 3,
                          background:
                            "linear-gradient(135deg, rgba(248,250,252,1), rgba(241,245,249,0.92))",
                          border: "1px solid rgba(148,163,184,0.18)",
                          cursor: o.url ? "pointer" : "default",
                          transition: "160ms ease",
                          "&:hover": o.url
                            ? {
                              transform: "translateY(-1px)",
                              background: "#eef6ff",
                              boxShadow: "0 12px 24px rgba(37,99,235,0.10)",
                            }
                            : undefined,
                        }}
                      >
                        <Stack direction="row" spacing={1.2} alignItems="center">
                          <Avatar sx={{ width: 28, height: 28 }}>
                            {o.name === "Яндекс Маркет" ? "Я" : o.name === "Ozon" ? "O" : "W"}
                          </Avatar>
                          <Typography sx={{ fontWeight: 700 }}>{o.name}</Typography>
                        </Stack>

                        <Typography sx={{ fontWeight: 800 }}>{o.price.toLocaleString("ru-RU")} ₽</Typography>
                      </Stack>
                    ))}
                  </Stack>

                  {/* Аналоги */}
                  <Accordion
                    disableGutters
                    elevation={0}
                    sx={{
                      borderRadius: 3,
                      overflow: "hidden",
                      border: "1px solid rgba(148,163,184,0.22)",
                      background: "linear-gradient(135deg, #ffffff, #f8fafc)",
                      "&:before": { display: "none" },
                    }}
                  >
                    <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                      <Typography sx={{ fontWeight: 800 }}>Варианты аналогов</Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                      <Stack spacing={1.4}>
                        {selectedInfo.analogs.map((a) => (
                          <Stack
                            key={a.name}
                            direction="row"
                            spacing={1.5}
                            alignItems="center"
                            justifyContent="space-between"
                            sx={{
                              p: 1,
                              borderRadius: 3,
                              background: "#fff",
                              border: "1px solid rgba(148,163,184,0.18)",
                            }}
                          >
                            <Stack direction="row" spacing={1.3} alignItems="center">
                              <Box
                                sx={{
                                  width: 56,
                                  height: 56,
                                  borderRadius: 2,
                                  overflow: "hidden",
                                  background: "#f0f2f6",
                                  flexShrink: 0,
                                }}
                              >
                                <img
                                  src={a.img}
                                  alt={a.name}
                                  style={{ width: "100%", height: "100%", objectFit: "cover", display: "block" }}
                                />
                              </Box>

                              <Box>
                                <Typography sx={{ fontWeight: 700, lineHeight: 1.2 }}>{a.name}</Typography>
                                <Typography variant="body2" color="text.secondary">
                                  {a.price.toLocaleString("ru-RU")} ₽
                                </Typography>
                              </Box>
                            </Stack>
                            <Button
                              variant="outlined"
                              size="small"
                              onClick={() => openUrl(a.url)}
                              disabled={!a.url}
                            >
                              Перейти
                            </Button>
                          </Stack>
                        ))}
                      </Stack>
                    </AccordionDetails>
                  </Accordion>

                  {/* Кнопки */}
                  <Stack direction="row" spacing={1.2}>
                    <Button
                      fullWidth
                      variant="contained"
                      onClick={() => openUrl(bestOffer?.url)}
                      disabled={!bestOffer?.url}
                      sx={{
                        fontWeight: 900,
                        borderRadius: 3,
                        py: 1.2,
                        background: "linear-gradient(135deg, #2563eb, #0ea5e9)",
                        boxShadow: "0 14px 28px rgba(37,99,235,0.24)",
                      }}
                    >
                      Открыть маркетплейс
                    </Button>

                    <Button
                      fullWidth
                      variant="outlined"
                      onClick={() => alert("Здесь будет подбор аналогов")}
                      sx={{
                        fontWeight: 900,
                        borderRadius: 3,
                        py: 1.2,
                        borderColor: "rgba(37,99,235,0.35)",
                        backgroundColor: "rgba(37,99,235,0.04)",
                      }}
                    >
                      Аналоги
                    </Button>
                  </Stack>
                </Stack>
              )}
            </CardContent>
          </Card>
        </Stack>
      </Box>
      <Dialog open={addOpen} onClose={() => setAddOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle sx={{ fontWeight: 800 }}>Добавить устройство</DialogTitle>

        <DialogContent sx={{ pt: 1 }}>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <FormControl fullWidth>
              <InputLabel>Тип</InputLabel>
              <Select
                value={newDeviceId}
                label="Тип"
                onChange={(e) => setNewDeviceId(String(e.target.value))}
              >
                <MenuItem value="lock">Замок</MenuItem>
                <MenuItem value="light">Свет</MenuItem>
                <MenuItem value="climate">Климат</MenuItem>
                <MenuItem value="perimeter">Датчик</MenuItem>
              </Select>
            </FormControl>

            <TextField
              label="Название"
              value={newDeviceName}
              onChange={(e) => setNewDeviceName(e.target.value)}
              fullWidth
            />

            <FormControl fullWidth>
              <InputLabel>Экосистема</InputLabel>
              <Select
                value={newDeviceEco}
                label="Экосистема"
                onChange={(e) => setNewDeviceEco(e.target.value as Ecosystem)}
              >
                <MenuItem value="Apple HomeKit">Apple HomeKit</MenuItem>
                <MenuItem value="Google Home">Google Home</MenuItem>
                <MenuItem value="Yandex Home">Yandex Home</MenuItem>
              </Select>
            </FormControl>

            <TextField
              label="Цена (₽)"
              type="number"
              value={newDevicePrice}
              onChange={(e) => setNewDevicePrice(Number(e.target.value))}
              fullWidth
            />
          </Stack>
        </DialogContent>

        <DialogActions sx={{ p: 2 }}>
          <Button variant="outlined" onClick={() => setAddOpen(false)}>
            Отмена
          </Button>

          <Button variant="contained" onClick={addDevice} sx={{ fontWeight: 800 }}>
            Добавить
          </Button>
        </DialogActions>
      </Dialog>

    </Box>
  );
}

function StatPill(props: { label: string; value: string; tone?: "success" | "danger" }) {
  const color = props.tone === "danger" ? "#dc2626" : props.tone === "success" ? "#16a34a" : "#2563eb";

  return (
    <Box
      sx={{
        minWidth: 118,
        px: 1.6,
        py: 1.1,
        borderRadius: 3,
        background: `linear-gradient(135deg, ${alpha(color, 0.12)}, ${alpha(color, 0.04)})`,
        border: `1px solid ${alpha(color, 0.18)}`,
      }}
    >
      <Typography variant="caption" sx={{ color: "#64748b", fontWeight: 800 }}>
        {props.label}
      </Typography>
      <Typography sx={{ color, fontWeight: 900, lineHeight: 1.15 }}>{props.value}</Typography>
    </Box>
  );
}
