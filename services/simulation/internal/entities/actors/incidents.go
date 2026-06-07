package actors

import (
	"encoding/json"
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

const (
	KindFireSpread = "fire:spread"
	fireSpreadRate = 0.5
)

type Fire struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[FireInData]

	ID           string   `json:"id"`
	X            float64  `json:"x"`
	Y            float64  `json:"y"`
	RoomID       string   `json:"roomID"`
	Receivers    []string `json:"receivers"`
	zones        []*FireZoneData
	burningRooms map[string]bool
}

type FireInData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
	Tick   bool   `json:"tick"`
}

type FireOutData struct {
	Kind  string          `json:"kind"`
	Fires []*FireZoneData `json:"fires"`
}

type FireSpreadPayload struct {
	Kind   string  `json:"kind"`
	RoomID string  `json:"roomID"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
}

type FireZoneData struct {
	RoomID string  `json:"roomID"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
}

func NewFire(data []byte, engineAPI engine.EnginePort) (*Fire, error) {
	var f Fire
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}

	f.enginePort = engineAPI
	f.inStore = *simgo.NewStore[FireInData](engineAPI.GetSimulation())
	f.burningRooms = map[string]bool{f.RoomID: true}

	return &f, nil
}

func (f *Fire) HandleInDTO(dto []byte) error {
	input := FireInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	f.inStore.Put(input)

	return nil
}

func (f *Fire) HandleOutDTO(dto []byte) {
	f.enginePort.GetOutChan() <- api.EventOutDTO{
		EntityID: f.ID,
		Payload:  dto,
	}

	for _, r := range f.Receivers {
		f.enginePort.GetInChan() <- api.EventInDTO{
			EntityID: r,
			Payload:  dto,
		}
	}
}

func (f *Fire) GetProcessFunc() func(simgo.Process) {
	return f.Process
}

func (f *Fire) Process(process simgo.Process) {
	for {
		el := f.inStore.Get()
		process.Wait(el.Event)
		if el.Item.TurnOn {
			break
		}
	}

	floor := f.enginePort.GetFloor()
	f.zones = []*FireZoneData{
		{
			RoomID: f.RoomID,
			X:      f.X,
			Y:      f.Y,
			Radius: 0,
		},
	}

	f.inStore.Put(FireInData{Tick: true})
	for {
		el := f.inStore.Get()
		process.Wait(el.Event)
		floor = f.enginePort.GetFloor()

		var newZones []*FireZoneData
		for _, zone := range f.zones {
			zone.Radius += fireSpreadRate

			f.notifyObserversInRoom(zone)

			for _, edge := range floor.Adjacency[zone.RoomID] {
				if edge.Door == nil {
					continue
				}

				neighborID := edge.NeighborRoomID
				if f.burningRooms[neighborID] {
					continue
				}

				door := edge.Door
				doorMidX := (door.Points[0][0] + door.Points[1][0]) / 2
				doorMidY := (door.Points[0][1] + door.Points[1][1]) / 2
				distToDoor := math.Sqrt((zone.X-doorMidX)*(zone.X-doorMidX) + (zone.Y-doorMidY)*(zone.Y-doorMidY))
				if zone.Radius < distToDoor {
					continue
				}

				f.burningRooms[neighborID] = true

				newZones = append(newZones, &FireZoneData{
					RoomID: neighborID,
					X:      doorMidX,
					Y:      doorMidY,
					Radius: 0,
				})
			}
		}
		f.zones = append(f.zones, newZones...)
		dto, err := json.Marshal(FireOutData{Kind: KindFireSpread, Fires: f.zones})
		if err != nil {
			return
		}

		f.HandleOutDTO(dto)
		f.inStore.Put(FireInData{Tick: true})
	}
}

func (f *Fire) notifyObserversInRoom(zone *FireZoneData) {
    dto, err := json.Marshal(FireSpreadPayload{
        Kind:   KindFireSpread,
        RoomID: zone.RoomID,
        X:      zone.X,
        Y:      zone.Y,
        Radius: zone.Radius,
    })
    if err != nil {
        return
    }

    for _, observerID := range f.enginePort.GetRoomObservers(zone.RoomID) {
        observer := f.enginePort.GetEntity(observerID).(entities.Observer)

        for _, k := range observer.GetObservedKinds() {
            if k == KindFireSpread {
                f.enginePort.GetInChan() <- api.EventInDTO{
                    EntityID: observerID,
                    Payload:  dto,
                }
                break
            }
        }
    }
}

func (f *Fire) GetID() string {
	return f.ID
}

func (f *Fire) GetReceiversID() []string {
	return f.Receivers
}

func (f *Fire) SetReceivers(actions []api.EdgeDTO) {
	f.Receivers = make([]string, len(actions))
	for i, a := range actions {
		f.Receivers[i] = a.ToID
	}
}
