package engine

type CustomDeviceInput struct {
	Device string `json:"device"`
	Count  int    `json:"count"`
}

type PriceInfo struct {
	MinPrice int
	MaxPrice int
}

type TriggerInfo struct {
	Description string   `json:"description"`
	Triggers    []string `json:"triggers"`
}
