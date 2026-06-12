package point

// Interval — отрезок на числовой оси стены [From, To)
type Interval struct {
	From float64
	To   float64
}

func (iv Interval) Length() float64 {
	return iv.To - iv.From
}
