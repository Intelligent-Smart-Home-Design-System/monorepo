package entities

type Apartment struct {
	ID string `json:"id"`
	Tracks []string `json:"tracks"`
	Rooms []*Room `json:"rooms"`
}

type ApartmentResult struct {
	Placements map[string]map[string]*Placement // roomID -> deviceID -> devicePlacement
												// То есть по roomID получаем мапу между
												// устройством и его расстановкой
	
	// в дальнейшем необходимо будет хранить доп поля в этой структуре (для других модулей)
}

func NewApartmentResult() *ApartmentResult {
	return &ApartmentResult{Placements: make(map[string]map[string]*Placement)}
}

type Room struct {
	ID string `json:"id"`
	Name string `json:"name"`
	WetPoints []*Point `json:"wet_points"`
}
