package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCenterMethod(t *testing.T) {
	room := Room{
		Name: "bathroom",
		Area: []Point{
			{X: 1, Y: 0},
			{X: 5, Y: 0},
			{X: 5, Y: 5},
			{X: 1, Y: 5},
		},
	}

	center, err := room.GetCenter()
	
	assert.NoError(t, err)
	assert.Equal(t, Point{X: 3, Y: 2.5}, *center)
}

func TestGetObjectCenterMethod(t *testing.T) {
	door := Door{
		Points: []Point{
			{X: 1, Y: 0},
			{X: 2, Y: 0},
		},
	}

	doorCenter := GetObjectCenter(door.Points)

	assert.Equal(t, Point{X: 1.5, Y: 0}, doorCenter)
}

func TestGridMethodSize(t *testing.T) {
	room := Room{
		Name: "kitchen",
		Area: []Point{
			{X: 0, Y: 0},
			{X: 3, Y: 0},
			{X: 3, Y: 3},
			{X: 0, Y: 3},
		},
	}

	step := 0.5
	gridPoints, err := room.GenerateGridPoints(step)

	assert.NoError(t, err)
	assert.Equal(t, int((3 / step) * (3 / step)), len(gridPoints))
}

func TestIsPointInRoomMethodPositive(t *testing.T) {
	room := Room{
		Name: "kitchen",
		Area: []Point{
			{X: 0, Y: 0},
			{X: 3, Y: 0},
			{X: 3, Y: 3},
			{X: 0, Y: 3},
		},
	}

	assert.Equal(t, true, room.IsPointInRoom(Point{X: 2, Y: 2.9}))
}

func TestIsPointInRoomMethodNegative(t *testing.T) {
	room := Room{
		Name: "living",
		Area: []Point{
			{X: 0, Y: 0},
			{X: 4, Y: 0},
			{X: 4, Y: 4},
			{X: 2, Y: 4},
			{X: 2, Y: 3},
			{X: 0, Y: 3},
		},
	}

	assert.Equal(t, false, room.IsPointInRoom(Point{X: 1, Y: 4}))
}

func TestIsWallBetweenPoints(t *testing.T) {
	room := Room{
		Name: "bathroom",
		Area: []Point{
			{X: 1, Y: 0},
			{X: 5, Y: 0},
			{X: 5, Y: 5},
			{X: 1, Y: 5},
		},
	}

	walls := []Wall{
		{
			ID: "1",
			Points: []Point{
				{X: 1, Y: 0},
				{X: 5, Y: 0},
			},
		},
		{
			ID: "2",
			Points: []Point{
				{X: 5, Y: 0},
				{X: 5, Y: 5},
			},
		},
		{
			ID: "3",
			Points: []Point{
				{X: 5, Y: 5},
				{X: 1, Y: 5},
			},
		},
	}

	apartment := &Apartment{
		Walls: walls,
		Rooms:   []Room{room},
	}
	apartment.MakeDependency()

	assert.Equal(t, false, apartment.IsWallBetweenPoints(Point{X: 2, Y: 3}, Point{X: 3, Y: 2}))
	assert.Equal(t, true, apartment.IsWallBetweenPoints(Point{X: 2, Y: 3}, Point{X: 6, Y: 2}))
}

func TestGetFrontDoorMethod(t *testing.T) {

	doors := []Door{
		{
			Points: []Point{
				{X: 1, Y: 0},
				{X: 2, Y: 0},
			},
			Rooms: []string{"1", "2"},
		},
		{
			Points: []Point{
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			},
			Rooms: []string{"2", "3"},
		},
		{
			Points: []Point{
				{X: 1, Y: 2},
				{X: 3, Y: 100},
			},
			Rooms: []string{"1"},
		},
	}

	apartment := Apartment{Doors: doors}
	frontDoor := Door{
		Points: []Point{
			{X: 1, Y: 2},
			{X: 3, Y: 100},
		},
		Rooms: []string{"1"},
	}

	assert.Equal(t, frontDoor, *apartment.GetFrontDoor())
}
