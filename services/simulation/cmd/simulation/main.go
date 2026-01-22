package main

import (
	"log"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/actors"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/field"
	"github.com/fschuetz04/simgo"
)

func main() {
	/////////////////////////////////
	//проверка работы пожара
	fld, err := field.Load("../../internal/field/simple_field.json")
	if err != nil {
		log.Fatal(err)
	}

	startCell := fld.GetCell(2, 2)

	sim := simgo.NewSimulation()

	fire := actors.NewFire("fire_1", fld, startCell, 1.0)

	triggerEvent := sim.Event()
	fire.SetTrigger(triggerEvent)

	sim.Process(fire.Process)

	sim.Process(func(p simgo.Process) {
		// подожди 0.1 единицы симвремени, затем триггерь.
		// (подкорректируй задержку если нужно)
		p.Wait(p.Timeout(0.1))
		triggerEvent.Trigger()
	})

	sim.Run()
	/////////////////////////////////
}
