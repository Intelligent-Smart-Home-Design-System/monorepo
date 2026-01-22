package field

import (
	"encoding/json"
	"os"
)

type Cell struct {
	X, Y  int
	Burnt bool
}

// это структура поля (в будущем - квартиры)
type Field struct {
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	RawCells [][]int `json:"cells"`

	Cells [][]*Cell
}

// загрузка содержимого json в структуру поля
func Load(path string) (*Field, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var field Field
	err = json.Unmarshal(data, &field)
	if err != nil {
		return nil, err
	}

	field.Cells = make([][]*Cell, field.Width)
	for x := 0; x < field.Width; x++ {
		field.Cells[x] = make([]*Cell, field.Height)
		for y := 0; y < field.Height; y++ {
			field.Cells[x][y] = &Cell{
				X:     x,
				Y:     y,
				Burnt: false,
			}
		}
	}
	return &field, nil
}

// Лучше если взаимодействие с полем будет происходить через функции
type GeneralField interface {
	GetCell(x, y int) *Cell
	GetNeighbors(c *Cell) []*Cell
}

func (f *Field) GetCell(x, y int) *Cell {
	if x < 0 || x >= f.Width || y < 0 || y >= f.Height {
		return nil
	}
	return f.Cells[x][y]
}

func (f *Field) GetNeighbors(c *Cell) []*Cell {
	directions := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	var result []*Cell

	for _, d := range directions {
		neighbor := f.GetCell(c.X+d[0], c.Y+d[1])
		if neighbor != nil {
			result = append(result, neighbor)
		}
	}
	return result
}
