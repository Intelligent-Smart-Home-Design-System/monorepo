package worker

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
