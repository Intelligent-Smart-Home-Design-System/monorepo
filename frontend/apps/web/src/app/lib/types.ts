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

export type ApiCatalogCategory = {
    id: string;
    name: string;
    product_count: number;
    min_price: number;
};

export type ApiCatalogProduct = {
    device_id: number;
    listing_id: number;
    device_type: string;
    name: string;
    brand: string;
    model: string;
    quality: number;
    price: number;
    currency: string;
    image_url?: string | null;
    rating: number;
    review_count: number;
    devices_per_listing: number;
    merchant: string;
    url: string;
    available: boolean;
    device_attributes: Record<string, unknown>;
    ecosystems: string[];
    protocols: string[];
};

export type ApiCatalogProductFilters = {
    brands: string[];
    ecosystems: string[];
    protocols: string[];
    min_price: number;
    max_price: number;
};

export type ApiCatalogProductsResponse = {
    items: ApiCatalogProduct[];
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
    filters: ApiCatalogProductFilters;
};

export type ApiCatalogProductsQuery = {
    device_type: string;
    q?: string;
    brand?: string;
    ecosystem?: string;
    protocol?: string;
    min_price?: number;
    max_price?: number;
    sort?: "price_asc" | "price_desc" | "rating_desc" | "name_asc";
    page?: number;
    page_size?: number;
};

export type ApiCreateManualPlanSelection = {
    device_id: number;
    listing_id: number;
    quantity: number;
};

export type ApiCreateManualPlanRequest = {
    budget: number;
    floor_plan: Record<string, unknown>;
    selections: ApiCreateManualPlanSelection[];
};

export type ApiCreatePlanResponse = {
    plan_id: number;
    status: string;
    message?: string;
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

export type ApiStartPipelineRequirement = {
    requirement_id: number;
    device_type: string;
    count: number;
    connect_to_main_ecosystem: boolean;
    filters?: ApiRequirementFilter[];
};

export type ApiStartPipelineRequest = {
    request_id?: string;
    floor_plan: Record<string, unknown>;
    selected_levels: Record<string, string>;
    device_selection: {
        main_ecosystem: string;
        budget: number;
        requirements: ApiStartPipelineRequirement[];
        max_solutions?: number;
        time_budget_seconds?: number;
    };
};

export type ApiStartPipelineResponse = {
    workflow_id: string;
    run_id?: string;
};

export type ApiPipelineResult = {
    request_id?: string;
    parsed_floor_plan?: unknown;
    layout?: unknown;
    dependencies?: Record<string, string[]> | null;
    device_selection?: unknown;
    stages?: ApiPlanStageArtifact[] | null;
    artifacts?: ApiPlanStageArtifact[] | null;
};

export type ApiPlanStatus = {
    plan_id: number;
    status: "queued" | "generating" | "completed" | "failed";
    progress?: number | null;
    stages?: ApiPlanStageArtifact[] | null;
    error?: ApiErrorResponse | null;
};

export type ApiPlanStageArtifact = {
    key: string;
    name?: string | null;
    title?: string | null;
    status?: "pending" | "running" | "completed" | "failed" | string | null;
    progress?: number | null;
    data?: unknown;
    payload?: unknown;
    updated_at?: string | null;
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
    device_quantity?: number;
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
    dependencies?: Record<string, string[]> | null;
    allowed_ecosystems?: string[] | null;
    excluded_ecosystems?: string[] | null;
    requirements: ApiRequirement[];
    bundles: ApiBundle[];
    stages?: ApiPlanStageArtifact[] | null;
    artifacts?: ApiPlanStageArtifact[] | null;
    floor_plan?: unknown;
};

export type ApiErrorResponse = {
    message: string;
    code?: string | null;
    details?: string | null;
};

export type AuthTokens = {
    access_token: string;
    refresh_token: string;
    token_type?: string;
};

export type AuthUser = {
    id: string;
    email?: string;
    name?: string | null;
};

export type LoginRequest = {
    email?: string;
    password?: string;
    is_authorising: boolean;
};

export type RegisterRequest = {
    email: string;
    password: string;
    name?: string;
};

export type RefreshTokenRequest = {
    refresh_token: string;
};

export type AuthResponse = AuthTokens & {
    user?: AuthUser | null;
    message?: string;
};
