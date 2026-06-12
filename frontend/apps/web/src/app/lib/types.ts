export type Ecosystem = string;

export type TrackKey = "security" | "light" | "climate" | "perimeter";

export type RequirementItem = {
    id: string;
    name: string;
    description: string;
    count: number;
    enabled: boolean;
};

export type TrackRequirement = {
    score: number;
    items: RequirementItem[];
};

export type RequirementsByTrack = Record<TrackKey, TrackRequirement>;

export type SceneAction = "on" | "off" | "auto";

export type SceneDeviceState = {
    deviceId: string;  // id устройства (из devices)
    action: SceneAction; // что сделать
};

export type Scene = {
    id: string;
    name: string;
    ecosystem: Ecosystem;  // чтобы сцены были привязаны к экосистеме
    items: SceneDeviceState[];
    createdAt: number;
    runCount: number; // на будущее для “популярности”
};

export type ApiFilterOperation =
    | "eq"
    | "neq"
    | "gt"
    | "gte"
    | "lt"
    | "lte"
    | "contains"
    | "exists";

export type ApiDeviceTypeFilterField = {
    name: string;
    field: string;
    value_type: "string" | "number" | "integer" | "boolean";
    enum_values?: string[] | null;
    operations: ApiFilterOperation[];
};

export type ApiDeviceType = {
    id: string;
    name: string;
    filters: ApiDeviceTypeFilterField[];
};

export type ApiRequirementFilter = {
    field: string;
    operation: ApiFilterOperation;
    value?: string | number | boolean | null;
};

export type ApiRequirement = {
    id: number;
    device_type: string;
    quantity: number;
    filters: ApiRequirementFilter[];
};

export type ApiPreset = {
    id: string;
    name: string;
    description?: string | null;
    requirements: ApiRequirement[];
};

export type ApiEcosystem = {
    id: string;
    name: string;
    description: string;
    may_be_main: boolean;
    image_url?: string | null;
};

export type ApiPlanSummary = {
    plan_id: number;
    created_at: string;
    budget: number;
    status: "queued" | "generating" | "completed" | "failed";
};

export type ApiCreatePlanRequest = {
    budget: number;
    main_ecosystem_id: string;
    allowed_ecosystems?: string[] | null;
    excluded_ecosystems?: string[] | null;
    requirements: Omit<ApiRequirement, "id">[];
};

export type ApiCreatePlanResponse = {
    plan_id: number;
    status: "accepted";
    message?: string;
};

export type ApiPlanStatus = {
    plan_id: number;
    status: "queued" | "generating" | "completed" | "failed";
    progress?: number | null;
    error?: ApiErrorResponse | null;
};

export type ApiConnectionInfo = {
    direct_ecosystem: string;
    direct_protocol: string;
    direct_hub_selected_listing_id?: number | null;
    direct_description?: string | null;
    final_ecosystem: string;
    final_protocol: string;
    final_hub_selected_listing_id?: number | null;
    final_description?: string | null;
};

export type ApiListing = {
    id: number;
    name: string;
    device_brand: string;
    device_model: string;
    device_quality_score: number;
    price: number;
    url: string;
    image_url?: string | null;
    devices_per_listing: number;
    units_to_buy: number;
    requirement_id: number;
    device_attributes?: Record<string, unknown>;
    connection_info: ApiConnectionInfo;
};

export type ApiBundle = {
    id: number;
    total_cost: number;
    quality_score: number;
    extra_ecosystems_used: number;
    hubs_used: number;
    is_recommended: boolean;
    ecosystems_used?: string[];
    listings: ApiListing[];
};

export type ApiHomePlan = {
    plan_id: number;
    budget: number;
    main_ecosystem_id: string;
    allowed_ecosystems?: string[] | null;
    excluded_ecosystems?: string[] | null;
    requirements: ApiRequirement[];
    bundles: ApiBundle[];
};

export type ApiErrorResponse = {
    message: string;
    code?: string | null;
    details?: string | null;
};
