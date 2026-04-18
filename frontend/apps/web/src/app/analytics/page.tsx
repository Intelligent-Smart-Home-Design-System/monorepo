"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import type { Ecosystem, TrackKey } from "../lib/types";
import {
    Box,
    Button,
    Card,
    CardContent,
    Divider,
    Stack,
    Typography,
    LinearProgress,
    Chip,
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
    budget: number;
    ecosystem: Ecosystem;
    hubs: Record<string, boolean>;
    tracks: Record<TrackKey, number>;
    fileName?: string;
};

export default function AnalyticsPage() {
    const router = useRouter();

    const [settings, setSettings] = useState<Settings | null>(null);
    const [devices, setDevices] = useState<Device[]>([]);

    // 1) Читаем settings (как в plan)
    useEffect(() => {
        const raw = localStorage.getItem("settings");
        if (!raw) {
            router.push("/settings");
            return;
        }
        try {
            setSettings(JSON.parse(raw));
        } catch {
            router.push("/settings");
        }
    }, [router]);

    useEffect(() => {
        const raw = localStorage.getItem("devices");
        if (!raw) return;
        try {
            setDevices(JSON.parse(raw));
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

    const total = useMemo(() => {
        return recommended.reduce((acc, x) => acc + x.device.price, 0);
    }, [recommended]);

    const left = settings ? settings.budget - total : 0;

    // 4) Разбивка бюджета по трекам (по “цене * доля вклада трека”)
    // Считаем вклад трека у устройства = weight * impact
    // Потом распределяем цену устройства пропорционально этим вкладом
    const breakdown = useMemo(() => {
        if (!settings) return null;

        const keys = Object.keys(settings.tracks) as TrackKey[];
        const rubByTrack: Record<TrackKey, number> = {} as any;
        const valueByTrack: Record<TrackKey, number> = {} as any;

        for (const k of keys) {
            rubByTrack[k] = 0;
            valueByTrack[k] = 0;
        }

        for (const item of recommended) {
            const d = item.device;

            // вклад по трекам
            const contrib: Record<TrackKey, number> = {} as any;
            let sumContrib = 0;

            for (const k of keys) {
                const w = settings.tracks[k] ?? 0;
                const impact = d.impacts[k] ?? 0;
                const c = w * impact;
                contrib[k] = c;
                sumContrib += c;
            }

            // если вкладов нет — пропускаем распределение
            if (sumContrib <= 0) continue;

            for (const k of keys) {
                const share = contrib[k] / sumContrib;
                rubByTrack[k] += d.price * share;
                valueByTrack[k] += contrib[k]; // “полезность”
            }
        }

        const sumRub = keys.reduce((acc, k) => acc + (rubByTrack[k] ?? 0), 0);
        const sumVal = keys.reduce((acc, k) => acc + (valueByTrack[k] ?? 0), 0);

        return { keys, rubByTrack, valueByTrack, sumRub, sumVal };
    }, [recommended, settings]);

    const labelOf: Record<string, string> = {
        security: "Безопасность",
        light: "Свет",
        climate: "Климат",
        perimeter: "Периметр",
    };

    if (!settings) {
        return <Box sx={{ minHeight: "100vh" }} />;
    }

    return (
        <Box sx={{ minHeight: "100vh", p: 4 }}>
            <Box sx={{ maxWidth: 1100, mx: "auto" }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
                    <Typography variant="h5" sx={{ fontWeight: 800, color: "#ffffff" }}>
                        Аналитика проекта
                    </Typography>

                    <Stack direction="row" spacing={1}>
                        <Button
                            variant="contained"
                            onClick={() => router.push("/plan")}
                            sx={{
                                background: "#fff",
                                color: "#1f2937",
                                fontWeight: 700,

                                boxShadow: "0 2px 6px rgba(0,0,0,0.08)",

                                "&:hover": {
                                    background: "#f9fafb",
                                    boxShadow: "0 4px 12px rgba(0,0,0,0.12)",
                                },
                            }}
                        >
                            Назад к плану
                        </Button>

                    </Stack>
                </Stack>

                <Card sx={{ borderRadius: 4, mb: 2 }}>
                    <CardContent>
                        <Stack
                            direction={{ xs: "column", md: "row" }}
                            justifyContent="space-between"
                            alignItems={{ xs: "flex-start", md: "center" }}
                            spacing={1}
                        >
                            <Stack spacing={0.3}>
                                <Typography sx={{ fontWeight: 800 }}>Итоги</Typography>
                                <Typography color="text.secondary">
                                    Экосистема: <b>{settings.ecosystem}</b>
                                </Typography>
                            </Stack>

                            <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                                <Typography>
                                    Бюджет: <b>{settings.budget.toLocaleString("ru-RU")} ₽</b>
                                </Typography>
                                <Typography>
                                    Набор: <b>{total.toLocaleString("ru-RU")} ₽</b>
                                </Typography>
                                <Typography color={left < 0 ? "error.main" : "text.primary"}>
                                    Остаток: <b>{left.toLocaleString("ru-RU")} ₽</b>
                                </Typography>
                            </Stack>
                        </Stack>

                        <Divider sx={{ my: 1.5 }} />

                        <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap" }}>
                            <Chip label="Рекомендованный набор" />
                            <Chip label={`${recommended.length} устройств`} />
                        </Stack>
                    </CardContent>
                </Card>

                <Card sx={{ borderRadius: 4 }}>
                    <CardContent>
                        <Typography sx={{ fontWeight: 800, mb: 1 }}>
                            Разбивка бюджета по трекам
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                            Цена устройств распределяется пропорционально их вкладу в треки (вес трека × impact).
                        </Typography>

                        {!breakdown ? (
                            <Typography color="text.secondary">Нет данных</Typography>
                        ) : (
                            <Stack spacing={1.6}>
                                {breakdown.keys.map((k) => {
                                    const rub = breakdown.rubByTrack[k] ?? 0;
                                    const pct = breakdown.sumRub > 0 ? (rub / breakdown.sumRub) * 100 : 0;

                                    return (
                                        <Box key={k}>
                                            <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.6 }}>
                                                <Typography sx={{ fontWeight: 700 }}>
                                                    {labelOf[k] ?? k}
                                                </Typography>
                                                <Typography sx={{ fontWeight: 800 }}>
                                                    {Math.round(rub).toLocaleString("ru-RU")} ₽ · {pct.toFixed(0)}%
                                                </Typography>
                                            </Stack>

                                            <LinearProgress variant="determinate" value={pct} />
                                        </Box>
                                    );
                                })}
                            </Stack>
                        )}
                    </CardContent>
                </Card>
            </Box>
        </Box>
    );
}
