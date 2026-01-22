package actors

// Пожар, протечка и тд

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/field"
	"github.com/fschuetz04/simgo"
)

type Fire struct {
	id         string
	field      field.GeneralField
	startCell  *field.Cell
	trigger    *simgo.Event
	spreadTime float64
}

func NewFire(id string, field field.GeneralField, start *field.Cell, spreadTime float64) *Fire {
	return &Fire{
		id:         id,
		field:      field,
		startCell:  start,
		spreadTime: spreadTime,
	}
}

// ////////////////////////////////
// реализация интерфейса Entity
func (f *Fire) SetEvent() {
}

func (f *Fire) Trigger(delay int) float64 {
	return 0
}

func (fire *Fire) Process(process simgo.Process) {
	for {
		if fire.trigger.Triggered() {
			fire.trigger = process.Simulation.Event()
		}

		process.Wait(fire.trigger)

		fmt.Printf(
			"Fire started at time %.1f in (%d,%d)",
			process.Simulation.Now(),
			fire.startCell.X,
			fire.startCell.Y,
		)

		fire.burn(process, fire.startCell)
	}
}

//////////////////////////////////

func (fire *Fire) burn(proc simgo.Process, cell *field.Cell) {
	if cell.Condition == 1 {
		return
	}
	cell.Condition = 1
	fmt.Printf(
		"Fire at (%d,%d), time %.1f",
		cell.X,
		cell.Y,
		proc.Simulation.Now(),
	)

	proc.Wait(proc.Timeout(fire.spreadTime))

	for _, n := range fire.field.GetNeighbors(cell) {
		if n.Condition == 0 {
			neighbor := n
			proc.Process(func(p simgo.Process) {
				fire.burn(p, neighbor)
			})
		}
	}
}
