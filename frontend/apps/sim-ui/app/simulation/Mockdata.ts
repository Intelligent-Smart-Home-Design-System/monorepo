export type LogLevel = "INFO" | "WARNING" | "ERROR";

export type LogEvent = {
  id: string;
  ts: string;
  level: LogLevel;
  device: string;
  message: string;
};

export type Scenario = {
  id: string;
  title: string;
  description?: string;
  chain: string[];
  category: "lighting" | "security" | "fire_gas" | "water" | "climate" | "comfort" | "energy" | "service";
};

export type DeviceStatus = "idle" | "active" | "error";

export type Device = {
  id: string;
  status: DeviceStatus;
};

export type Room = {
  id: string;
  title: string;
  x: number;
  y: number;
  w: number;
  h: number;
  labelX?: number;
  labelY?: number;
};


export const initialDevices: Device[] = [
  { id: "motion_sensor_hall", status: "idle" },
  { id: "hub", status: "idle" },
  { id: "lamp_hall", status: "idle" },
];

export const rooms: Room[] = [
  { id: "kitchen", title: "кухня", x: 0.68, y: 0.15, w: 0.30, h: 0.35 },
  { id: "bedroom", title: "спальня", x: 0.04, y: 0.35, w: 0.30, h: 0.40 },
  { id: "bedroom_2", title: "спальня 2", x: 0.04, y: 0.05, w: 0.30, h: 0.40 },
  { id: "living", title: "гостиная", x: 0.25, y: 0.15, w: 0.56, h: 0.45 },
  { id: "hall", title: "прихожая", x: 0.10, y: 0.59, w: 0.56, h: 0.45 },
  { id: "bath", title: "ванная", x: 0.72, y: 0.60, w: 0.24, h: 0.34 },
];

export const deviceMarkers: DeviceMarker[] = [
  { id: "hub", x: 0.52, y: 0.28 },
  { id: "lamp_hall", x: 0.30, y: 0.62 },
  { id: "motion_sensor_hall", x: 0.50, y: 0.80 },
];

export const scenarios: Scenario[] = [
  { id: "scn_1", title: "Движение → включить свет в прихожей", description: "Если темно", chain: ["motion_sensor_hall", "hub", "lamp_hall"], category: "lighting" },
  { id: "scn_2", title: "Нет движения 5 минут → выключить свет", description: "Прихожая", chain: ["motion_sensor_hall", "controller", "lamp_hall"], category: "lighting" },
  { id: "scn_3", title: "Открыли дверь → включить свет на 2 мин", description: "Прихожая", chain: ["door_sensor", "gateway", "lamp_hall"], category: "lighting" },
  { id: "scn_4", title: "Ночной режим → свет 20%", description: "Приглушить", chain: ["lux_sensor", "hub", "lamp_hall"], category: "lighting" },
  { id: "scn_5", title: "TV режим → приглушить свет", description: "Гостиная", chain: ["scene_button", "controller", "lamp_living"], category: "lighting" },

  { id: "scn_6", title: "Охрана: дверь открыта → сирена", description: "Тревога", chain: ["door_sensor", "hub", "siren"], category: "security" },
  { id: "scn_7", title: "Охрана: движение ночью → запись камеры", description: "Коридор", chain: ["motion_sensor_hall", "controller", "camera"], category: "security" },
  { id: "scn_8", title: "Вибрация окна → уведомление", description: "Сигнал", chain: ["vibration_sensor", "gateway", "notification"], category: "security" },
  { id: "scn_9", title: "Замок открыт слишком долго → уведомление", description: "Контроль", chain: ["lock_state", "hub", "notification"], category: "security" },

  { id: "scn_10", title: "Дым → сирена + свет", description: "Эвакуация", chain: ["smoke_sensor", "hub", "siren"], category: "fire_gas" },
  { id: "scn_11", title: "CO → вентиляция", description: "Проветривание", chain: ["co_sensor", "controller", "ventilation"], category: "fire_gas" },
  { id: "scn_12", title: "Газ → уведомление", description: "Осторожно", chain: ["gas_sensor", "gateway", "notification"], category: "fire_gas" },

  { id: "scn_13", title: "Протечка → перекрыть воду", description: "Авария", chain: ["leak_sensor", "hub", "water_valve"], category: "water" },
  { id: "scn_14", title: "Протечка в ванной → сирена", description: "Тревога", chain: ["leak_sensor_bath", "controller", "siren"], category: "water" },
  { id: "scn_15", title: "Расход воды ночью > порога → уведомление", description: "Подозрение", chain: ["water_flow", "gateway", "notification"], category: "water" },

  { id: "scn_16", title: "Температура низкая → обогрев", description: "Климат", chain: ["temp_sensor", "hub", "heater"], category: "climate" },
  { id: "scn_17", title: "Температура высокая → кондиционер", description: "Климат", chain: ["temp_sensor", "controller", "ac"], category: "climate" },
  { id: "scn_18", title: "Влажность высокая → вытяжка", description: "Ванная", chain: ["humidity_sensor", "gateway", "ventilation"], category: "climate" },
  { id: "scn_19", title: "CO2 высокий → проветривание", description: "Качество воздуха", chain: ["co2_sensor", "hub", "ventilation"], category: "climate" },
  { id: "scn_20", title: "Окно открыто → пауза отопления", description: "Экономия", chain: ["window_sensor", "controller", "heater"], category: "climate" },

  { id: "scn_21", title: "Утро → плавный свет", description: "Рутина", chain: ["scene_button", "hub", "lamp_living"], category: "comfort" },
  { id: "scn_22", title: "Пришли домой → свет в прихожей", description: "Комфорт", chain: ["presence_sensor", "gateway", "lamp_hall"], category: "comfort" },
  { id: "scn_23", title: "Ушли → выключить всё", description: "Режим отсутствия", chain: ["presence_sensor", "controller", "all_off"], category: "comfort" },
  { id: "scn_24", title: "Кнопка у кровати → всё выключить", description: "Ночь", chain: ["scene_button", "hub", "all_off"], category: "comfort" },
  { id: "scn_25", title: "TV включился → шторы + свет", description: "Кино", chain: ["tv_state", "controller", "curtains"], category: "comfort" },

  { id: "scn_26", title: "Нет присутствия → отключить розетки", description: "Экономия", chain: ["presence_sensor", "hub", "smart_plug"], category: "energy" },
  { id: "scn_27", title: "Пик потребления → отключить второстепенные", description: "Экономия", chain: ["power_meter", "controller", "load_shed"], category: "energy" },
  { id: "scn_28", title: "Ночной тариф → нагрев воды", description: "Экономия", chain: ["tariff_sensor", "gateway", "water_heater"], category: "energy" },

  { id: "scn_29", title: "Батарея низкая → уведомление", description: "Сервис", chain: ["battery_low", "hub", "notification"], category: "service" },
  { id: "scn_30", title: "Устройство оффлайн → уведомление", description: "Сервис", chain: ["device_offline", "controller", "notification"], category: "service" }
];

export type Floor = {
  version: 1;
  id: string;
  title: string;
  canvas?: { aspect?: number };
  rooms: Room[];
  markers: DeviceMarker[];
};

export type DeviceMarker = {
  id: string;
  x: number;
  y: number;
  label?: string;
};

export const floor: Floor = {
  version: 1,
  id: "flat_1",
  title: "Квартира",
  canvas: { aspect: 2.2 },
  rooms,
  markers: deviceMarkers,
};
