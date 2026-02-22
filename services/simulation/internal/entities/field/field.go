package field

type Cell struct {
	Condition    int  // пусть 1 - сгорело; 0 - дефолт
	IsHiddenWall bool // невидимая стенка для пожара/...
}

type Field struct {
	Width  int
	Height int
	Cells  [][]*Cell
}

func NewField(width, height int, cells [][]*Cell) *Field {
	return &Field{
		Width:  width,
		Height: height,
		Cells:  cells,
	}
}
