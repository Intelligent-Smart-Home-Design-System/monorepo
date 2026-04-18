"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import type { Ecosystem, TrackKey, Scene, SceneAction } from "../lib/types";
import {
    Box,
    Button,
    Card,
    CardContent,
    Divider,
    Stack,
    Typography,
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    FormControl,
    InputLabel,
    Select,
    MenuItem,
    Checkbox,
    FormControlLabel,
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

const LS_SCENES = "scenes";

function loadScenes(): Scene[] {
    if (typeof window === "undefined") return [];
    try {
        const raw = localStorage.getItem(LS_SCENES);
        if (!raw) return [];
        const parsed = JSON.parse(raw);
        return Array.isArray(parsed) ? (parsed as Scene[]) : [];
    } catch {
        return [];
    }
}

function saveScenes(next: Scene[]) {
    localStorage.setItem(LS_SCENES, JSON.stringify(next));
}

export default function ScenesPage() {
    const router = useRouter();

    const [settings, setSettings] = useState<Settings | null>(null);
    const [devices, setDevices] = useState<Device[]>([]);
    const [scenes, setScenes] = useState<Scene[]>(() => loadScenes());
    const [selectedSceneId, setSelectedSceneId] = useState<string | null>(null);

    // dialog create
    const [createOpen, setCreateOpen] = useState(false);
    const [newName, setNewName] = useState("Новая сцена");
    const [newEco, setNewEco] = useState<Ecosystem>("Apple HomeKit");

    // load settings/devices/scenes
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
        try {
            const raw = localStorage.getItem("devices");
            if (raw) setDevices(JSON.parse(raw));
        } catch { }
    }, []);

    // persist scenes
    useEffect(() => {
        if (typeof window === "undefined") return;
        saveScenes(scenes);
    }, [scenes]);

    const selectedScene = useMemo(() => {
        if (!selectedSceneId) return null;
        return scenes.find((s) => s.id === selectedSceneId) ?? null;
    }, [scenes, selectedSceneId]);

    const devicesForEco = useMemo(() => {
        const eco = selectedScene?.ecosystem ?? settings?.ecosystem;
        if (!eco) return [];
        return devices.filter((d) => d.ecosystem === eco);
    }, [devices, selectedScene?.ecosystem, settings?.ecosystem]);

    const openCreate = () => {
        setNewName("Новая сцена");
        setNewEco(settings?.ecosystem ?? "Apple HomeKit");
        setCreateOpen(true);
    };

    const createScene = () => {
        const id = `scene-${Date.now()}`;
        const scene: Scene = {
            id,
            name: newName.trim() || "Новая сцена",
            ecosystem: newEco,
            items: [],
            createdAt: Date.now(),
            runCount: 0,
        };
        setScenes((prev) => [scene, ...prev]);
        setSelectedSceneId(id);
        setCreateOpen(false);
    };

    const deleteScene = (id: string) => {
        setScenes((prev) => prev.filter((s) => s.id !== id));
        if (selectedSceneId === id) setSelectedSceneId(null);
    };

    const toggleDeviceInScene = (deviceId: string, checked: boolean) => {
        if (!selectedScene) return;

        setScenes((prev) =>
            prev.map((s) => {
                if (s.id !== selectedScene.id) return s;

                const exists = s.items.find((it) => it.deviceId === deviceId);
                if (checked && !exists) {
                    return { ...s, items: [...s.items, { deviceId, action: "on" }] };
                }
                if (!checked && exists) {
                    return { ...s, items: s.items.filter((it) => it.deviceId !== deviceId) };
                }
                return s;
            })
        );
    };

    const setAction = (deviceId: string, action: SceneAction) => {
        if (!selectedScene) return;

        setScenes((prev) =>
            prev.map((s) => {
                if (s.id !== selectedScene.id) return s;
                return {
                    ...s,
                    items: s.items.map((it) => (it.deviceId === deviceId ? { ...it, action } : it)),
                };
            })
        );
    };

    const renameScene = (name: string) => {
        if (!selectedScene) return;
        setScenes((prev) => prev.map((s) => (s.id === selectedScene.id ? { ...s, name } : s)));
    };

    if (!settings) return <Box sx={{ minHeight: "100vh" }} />;

    return (
        <Box sx={{ minHeight: "100vh", p: 4 }}>
            <Box sx={{ maxWidth: 1100, mx: "auto" }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
                    <Typography variant="h5" sx={{ fontWeight: 800, color: "#fff" }}>
                        Сценарии
                    </Typography>

                    <Stack direction="row" spacing={1}>
                        <Button variant="contained" onClick={openCreate} sx={{ fontWeight: 800 }}>
                            + Создать сцену
                        </Button>

                        <Button
                            variant="contained"
                            onClick={() => router.push("/plan")}
                            sx={{
                                background: "#fff",
                                color: "#1f2937",
                                fontWeight: 700,
                                boxShadow: "0 2px 6px rgba(0,0,0,0.08)",
                                "&:hover": { background: "#f9fafb" },
                            }}
                        >
                            Назад к плану
                        </Button>
                    </Stack>
                </Stack>

                <Stack direction={{ xs: "column", md: "row" }} spacing={2}>
                    {/* left: list */}
                    <Card sx={{ width: { xs: "100%", md: 360 }, borderRadius: 4 }}>
                        <CardContent>
                            <Typography sx={{ fontWeight: 900, mb: 1 }}>Мои сцены</Typography>
                            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                Сцены привязаны к экосистеме. Твоя текущая: <b>{settings.ecosystem}</b>
                            </Typography>

                            <Stack spacing={1}>
                                {scenes.length === 0 ? (
                                    <Typography color="text.secondary">Пока нет сценариев</Typography>
                                ) : (
                                    scenes.map((s) => {
                                        const isSel = s.id === selectedSceneId;
                                        return (
                                            <Box
                                                key={s.id}
                                                onClick={() => setSelectedSceneId(s.id)}
                                                sx={{
                                                    p: 1.2,
                                                    borderRadius: 2,
                                                    border: "1px solid rgba(0,0,0,0.08)",
                                                    background: isSel ? "#eef2ff" : "#fff",
                                                    cursor: "pointer",
                                                }}
                                            >
                                                <Stack direction="row" justifyContent="space-between" alignItems="center">
                                                    <Box>
                                                        <Typography sx={{ fontWeight: 900, lineHeight: 1.2 }}>{s.name}</Typography>
                                                        <Typography variant="body2" color="text.secondary">
                                                            {s.ecosystem} · {s.items.length} устройств
                                                        </Typography>
                                                    </Box>

                                                    <Button
                                                        variant="text"
                                                        color="error"
                                                        onClick={(e) => {
                                                            e.preventDefault();
                                                            e.stopPropagation();
                                                            deleteScene(s.id);
                                                        }}
                                                        sx={{ fontWeight: 800 }}
                                                    >
                                                        Удалить
                                                    </Button>
                                                </Stack>
                                            </Box>
                                        );
                                    })
                                )}
                            </Stack>
                        </CardContent>
                    </Card>

                    {/* right: editor */}
                    <Card sx={{ flex: 1, borderRadius: 4 }}>
                        <CardContent>
                            {!selectedScene ? (
                                <Typography color="text.secondary">
                                    Выбери сцену слева или создай новую.
                                </Typography>
                            ) : (
                                <Stack spacing={2}>
                                    <Stack direction={{ xs: "column", sm: "row" }} spacing={2} alignItems={{ sm: "center" }}>
                                        <Box sx={{ flex: 1 }}>
                                            <Typography sx={{ fontWeight: 900, mb: 0.6 }}>Название</Typography>
                                            <TextField
                                                value={selectedScene.name}
                                                onChange={(e) => renameScene(e.target.value)}
                                                fullWidth
                                            />
                                        </Box>

                                        <Box sx={{ minWidth: 260 }}>
                                            <Typography sx={{ fontWeight: 900, mb: 0.6 }}>Экосистема</Typography>
                                            <Chip label={selectedScene.ecosystem} />
                                        </Box>
                                    </Stack>

                                    <Divider />

                                    <Typography sx={{ fontWeight: 900 }}>Устройства в сцене</Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        Отметь устройства и выбери действие.
                                    </Typography>

                                    <Stack spacing={1}>
                                        {devicesForEco.length === 0 ? (
                                            <Typography color="text.secondary">
                                                Нет устройств для экосистемы <b>{selectedScene.ecosystem}</b>
                                            </Typography>
                                        ) : (
                                            devicesForEco.map((d) => {
                                                const inScene = selectedScene.items.find((it) => it.deviceId === d.id);
                                                const action: SceneAction = inScene?.action ?? "on";

                                                return (
                                                    <Box
                                                        key={d.id}
                                                        sx={{
                                                            p: 1.2,
                                                            borderRadius: 2,
                                                            border: "1px solid rgba(0,0,0,0.08)",
                                                            background: "#fff",
                                                        }}
                                                    >
                                                        <Stack
                                                            direction={{ xs: "column", sm: "row" }}
                                                            spacing={1.2}
                                                            alignItems={{ sm: "center" }}
                                                            justifyContent="space-between"
                                                        >
                                                            <FormControlLabel
                                                                control={
                                                                    <Checkbox
                                                                        checked={!!inScene}
                                                                        onChange={(e) => toggleDeviceInScene(d.id, e.target.checked)}
                                                                    />
                                                                }
                                                                label={
                                                                    <Box>
                                                                        <Typography sx={{ fontWeight: 800, lineHeight: 1.2 }}>{d.name}</Typography>
                                                                        <Typography variant="body2" color="text.secondary">
                                                                            {d.type} · {d.price.toLocaleString("ru-RU")} ₽
                                                                        </Typography>
                                                                    </Box>
                                                                }
                                                            />

                                                            <FormControl sx={{ minWidth: 180 }} disabled={!inScene}>
                                                                <InputLabel>Действие</InputLabel>
                                                                <Select
                                                                    value={action}
                                                                    label="Действие"
                                                                    onChange={(e) => setAction(d.id, e.target.value as SceneAction)}
                                                                >
                                                                    <MenuItem value="on">Включить</MenuItem>
                                                                    <MenuItem value="off">Выключить</MenuItem>
                                                                    <MenuItem value="auto">Авто</MenuItem>
                                                                </Select>
                                                            </FormControl>
                                                        </Stack>
                                                    </Box>
                                                );
                                            })
                                        )}
                                    </Stack>
                                </Stack>
                            )}
                        </CardContent>
                    </Card>
                </Stack>
            </Box>

            {/* create dialog */}
            <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="sm">
                <DialogTitle sx={{ fontWeight: 900 }}>Создать сцену</DialogTitle>
                <DialogContent sx={{ pt: 1 }}>
                    <Stack spacing={2} sx={{ mt: 1 }}>
                        <TextField
                            label="Название"
                            value={newName}
                            onChange={(e) => setNewName(e.target.value)}
                            fullWidth
                        />

                        <FormControl fullWidth>
                            <InputLabel>Экосистема</InputLabel>
                            <Select
                                value={newEco}
                                label="Экосистема"
                                onChange={(e) => setNewEco(e.target.value as Ecosystem)}
                            >
                                <MenuItem value="Apple HomeKit">Apple HomeKit</MenuItem>
                                <MenuItem value="Google Home">Google Home</MenuItem>
                                <MenuItem value="Yandex Home">Yandex Home</MenuItem>
                            </Select>
                        </FormControl>

                        <Typography variant="body2" color="text.secondary">
                            Сцена будет показывать устройства только выбранной экосистемы.
                        </Typography>
                    </Stack>
                </DialogContent>
                <DialogActions sx={{ p: 2 }}>
                    <Button variant="outlined" onClick={() => setCreateOpen(false)}>
                        Отмена
                    </Button>
                    <Button variant="contained" onClick={createScene} sx={{ fontWeight: 900 }}>
                        Создать
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}
