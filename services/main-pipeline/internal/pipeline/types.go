package pipeline

import "time"

const WorkflowName = "main_pipeline"

const (
	FloorParserActivityName     = "floor_parser.parse_floor"
	LayoutActivityName          = "layout.build_layout"
	DeviceSelectionActivityName = "device_selection.select_devices_from_file"
)

type Request struct {
	RequestID       string                 `json:"request_id"`
	FloorParser     *FloorParserStep       `json:"floor_parser,omitempty"`
	Layout          *LayoutStep            `json:"layout,omitempty"`
	DeviceSelection *DeviceSelectionStep   `json:"device_selection,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type TaskQueues struct {
	Orchestration   string `json:"orchestration"`
	FloorParser     string `json:"floor_parser"`
	Layout          string `json:"layout"`
	DeviceSelection string `json:"device_selection"`
}

type RetrySettings struct {
	MaximumAttempts    int32         `json:"maximum_attempts"`
	InitialInterval    time.Duration `json:"initial_interval"`
	BackoffCoefficient float64       `json:"backoff_coefficient"`
	MaximumInterval    time.Duration `json:"maximum_interval"`
}

type ActivitySettings struct {
	StartToCloseTimeout time.Duration `json:"start_to_close_timeout"`
	HeartbeatTimeout    time.Duration `json:"heartbeat_timeout"`
	Retry               RetrySettings `json:"retry"`
}

type WorkflowInput struct {
	Request
	TaskQueues TaskQueues       `json:"task_queues"`
	Activity   ActivitySettings `json:"activity"`
}

type FloorParserStep struct {
	SourcePath string `json:"source_path"`
	OutputPath string `json:"output_path"`
}

type LayoutStep struct {
	ApartmentPath  string            `json:"apartment_path"`
	OutputPath     string            `json:"output_path"`
	SelectedLevels map[string]string `json:"selected_levels"`
}

type DeviceSelectionStep struct {
	RequestPath string `json:"request_path"`
	OutputPath  string `json:"output_path"`
}

type FloorParserActivityInput struct {
	RequestID  string `json:"request_id"`
	SourcePath string `json:"source_path"`
	OutputPath string `json:"output_path"`
}

type FloorParserActivityOutput struct {
	RequestID    string `json:"request_id"`
	OutputPath   string `json:"output_path"`
	WallCount    int    `json:"wall_count"`
	DoorCount    int    `json:"door_count"`
	WindowCount  int    `json:"window_count"`
	WarningCount int    `json:"warning_count"`
}

type LayoutActivityInput struct {
	RequestID      string            `json:"request_id"`
	ApartmentPath  string            `json:"apartment_path"`
	OutputPath     string            `json:"output_path"`
	SelectedLevels map[string]string `json:"selected_levels"`
}

type LayoutActivityOutput struct {
	RequestID      string `json:"request_id"`
	OutputPath     string `json:"output_path"`
	PlacementCount int    `json:"placement_count"`
	MinPrice       int    `json:"min_price"`
	MaxPrice       int    `json:"max_price"`
}

type DeviceSelectionActivityInput struct {
	RequestID   string `json:"request_id"`
	RequestPath string `json:"request_path"`
	OutputPath  string `json:"output_path"`
}

type DeviceSelectionActivityOutput struct {
	RequestID     string  `json:"request_id"`
	OutputPath    string  `json:"output_path"`
	SolutionCount int     `json:"solution_count"`
	BestTotalCost float64 `json:"best_total_cost"`
	CatalogSource string  `json:"catalog_source"`
}

type WorkflowResult struct {
	RequestID       string                         `json:"request_id"`
	StartedAt       time.Time                      `json:"started_at"`
	CompletedAt     time.Time                      `json:"completed_at"`
	FloorParser     *FloorParserActivityOutput     `json:"floor_parser,omitempty"`
	Layout          *LayoutActivityOutput          `json:"layout,omitempty"`
	DeviceSelection *DeviceSelectionActivityOutput `json:"device_selection,omitempty"`
}
