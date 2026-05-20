package point

type Point struct {
	X float64
	Y float64
}

type Segment struct {
	LeftPoint  Point
	RightPoint Point
}

func NewVector(p1, p2 Point) Point {
	return Point{p2.X - p1.X, p2.Y - p1.Y}
}
