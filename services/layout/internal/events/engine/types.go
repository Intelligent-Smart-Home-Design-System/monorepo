package engine

type PriceInfo struct {
	MinPrice int
	MaxPrice int
}

type TriggerInfo struct {
	Description string   `json:"description"`
	Triggers    []string `json:"triggers"`
}
