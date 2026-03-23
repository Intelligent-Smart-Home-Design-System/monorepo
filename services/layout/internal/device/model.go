package device

type Device struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // Типы должны совпадать с классификацией
	DeviceTrack string `json:"device_tracks"`
}

// Placement представляет расстановку устройства
type Placement struct {
	Device *Device
	RoomID string
	Place  *Point
}

func NewDevice(ID, deviceType, trackType string) *Device {
	return &Device{ID: ID, Type: deviceType, DeviceTrack: trackType}
}

func NewPlacement(device *Device, roomID string, place Point) *Placement {
	return &Placement{Device: device, RoomID: roomID, Place: place}
}
