<<<<<<< HEAD
package geometry

import (
	"math"
	"sort"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type WallIntervalTracker struct {
	wallLen   float64
	blocked   []point.Interval // заблокировано зонами
	protected []point.Interval // защищено стенами (вычитается из blocked)
}

func NewWallIntervalTracker(wallLen float64) *WallIntervalTracker {
	return &WallIntervalTracker{
		wallLen:   wallLen,
		blocked:   make([]point.Interval, 0),
		protected: make([]point.Interval, 0),
	}
}

func (t *WallIntervalTracker) Block(iv point.Interval) {
	from := math.Max(0, iv.From)
	to := math.Min(t.wallLen, iv.To)
	if from >= to {
		return
	}
	t.blocked = append(t.blocked, point.Interval{From: from, To: to})
}

// Protect — стена между мебелью и нашей стеной, этот участок свободен
func (t *WallIntervalTracker) Protect(iv point.Interval) {
	from := math.Max(0, iv.From)
	to := math.Min(t.wallLen, iv.To)
	if from >= to {
		return
	}
	t.protected = append(t.protected, point.Interval{From: from, To: to})
}

// FreeIntervals вычисляет доступные (свободные) участки на стене для размещения объектов.
// Он учитывает занятые зоны (blocked) и защищенные области (protected), которые разрешено перекрывать.
// Возвращаемые интервалы гарантированно отсортированы по возрастанию координаты From.
func (t *WallIntervalTracker) FreeIntervals(minLen float64) []point.Interval {
	mergedBlocked := mergeIntervals(t.blocked)

	realBlocked := subtractProtected(mergedBlocked, mergeIntervals(t.protected))

	free := make([]point.Interval, 0)
	cursor := 0.0

	for _, b := range realBlocked {
		if b.From > cursor {
			iv := point.Interval{From: cursor, To: b.From}
			if iv.Length() >= minLen {
				free = append(free, iv)
			}
		}
		cursor = math.Max(cursor, b.To)
	}

	if cursor < t.wallLen {
		iv := point.Interval{From: cursor, To: t.wallLen}
		if iv.Length() >= minLen {
			free = append(free, iv)
		}
	}

	return free
}

// mergeIntervals сортирует и мёржит перекрывающиеся интервалы
func mergeIntervals(intervals []point.Interval) []point.Interval {
	if len(intervals) == 0 {
		return nil
	}

	sorted := make([]point.Interval, len(intervals))
	copy(sorted, intervals)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].From < sorted[j].From
	})

	merged := []point.Interval{sorted[0]}
	for _, iv := range sorted[1:] {
		last := &merged[len(merged)-1]
		if iv.From <= last.To {
			last.To = math.Max(last.To, iv.To)
		} else {
			merged = append(merged, iv)
		}
	}
	return merged
}

// subtractProtected вычитает protected из blocked.
// Оба списка должны быть уже смёрженными и отсортированными.
func subtractProtected(blocked, protected []point.Interval) []point.Interval {
	result := make([]point.Interval, 0)
	pi := 0

	for _, b := range blocked {
		cursor := b.From

		for pi < len(protected) && protected[pi].From < b.To {
			p := protected[pi]

			if p.To <= cursor {
				pi++
				continue
			}

			if p.From > cursor {
				result = append(result, point.Interval{From: cursor, To: p.From})
			}

			cursor = math.Max(cursor, p.To)
			pi++
		}

		if cursor < b.To {
			result = append(result, point.Interval{From: cursor, To: b.To})
		}
	}

	return result
}
=======
package geometry

import (
	"math"
	"sort"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type WallIntervalTracker struct {
	wallLen   float64
	blocked   []point.Interval // заблокировано зонами
	protected []point.Interval // защищено стенами (вычитается из blocked)
}

func NewWallIntervalTracker(wallLen float64) *WallIntervalTracker {
	return &WallIntervalTracker{
		wallLen:   wallLen,
		blocked:   make([]point.Interval, 0),
		protected: make([]point.Interval, 0),
	}
}

func (t *WallIntervalTracker) Block(iv point.Interval) {
	from := math.Max(0, iv.From)
	to := math.Min(t.wallLen, iv.To)
	if from >= to {
		return
	}
	t.blocked = append(t.blocked, point.Interval{From: from, To: to})
}

// Protect — стена между мебелью и нашей стеной, этот участок свободен
func (t *WallIntervalTracker) Protect(iv point.Interval) {
	from := math.Max(0, iv.From)
	to := math.Min(t.wallLen, iv.To)
	if from >= to {
		return
	}
	t.protected = append(t.protected, point.Interval{From: from, To: to})
}

// FreeIntervals вычисляет доступные (свободные) участки на стене для размещения объектов.
// Он учитывает занятые зоны (blocked) и защищенные области (protected), которые разрешено перекрывать.
// Возвращаемые интервалы гарантированно отсортированы по возрастанию координаты From.
func (t *WallIntervalTracker) FreeIntervals(minLen float64) []point.Interval {
	// 1. мёржим blocked
	mergedBlocked := mergeIntervals(t.blocked)

	// 2. вычитаем protected из blocked
	//    результат — реально заблокированные участки
	realBlocked := subtractProtected(mergedBlocked, mergeIntervals(t.protected))

	// 3. sweep — строим свободные из дырок
	free := make([]point.Interval, 0)
	cursor := 0.0

	for _, b := range realBlocked {
		if b.From > cursor {
			iv := point.Interval{From: cursor, To: b.From}
			if iv.Length() >= minLen {
				free = append(free, iv)
			}
		}
		cursor = math.Max(cursor, b.To)
	}

	if cursor < t.wallLen {
		iv := point.Interval{From: cursor, To: t.wallLen}
		if iv.Length() >= minLen {
			free = append(free, iv)
		}
	}

	return free
}

// mergeIntervals сортирует и мёржит перекрывающиеся интервалы
func mergeIntervals(intervals []point.Interval) []point.Interval {
	if len(intervals) == 0 {
		return nil
	}

	sorted := make([]point.Interval, len(intervals))
	copy(sorted, intervals)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].From < sorted[j].From
	})

	merged := []point.Interval{sorted[0]}
	for _, iv := range sorted[1:] {
		last := &merged[len(merged)-1]
		if iv.From <= last.To {
			last.To = math.Max(last.To, iv.To)
		} else {
			merged = append(merged, iv)
		}
	}
	return merged
}

// subtractProtected вычитает protected из blocked.
// Оба списка должны быть уже смёрженными и отсортированными.
func subtractProtected(blocked, protected []point.Interval) []point.Interval {
	result := make([]point.Interval, 0)
	pi := 0

	for _, b := range blocked {
		cursor := b.From

		for pi < len(protected) && protected[pi].From < b.To {
			p := protected[pi]

			if p.To <= cursor {
				pi++
				continue
			}

			// часть до защищённого — реально заблокирована
			if p.From > cursor {
				result = append(result, point.Interval{From: cursor, To: p.From})
			}

			cursor = math.Max(cursor, p.To)
			pi++
		}

		// остаток после всех защищённых
		if cursor < b.To {
			result = append(result, point.Interval{From: cursor, To: b.To})
		}
	}

	return result
}
>>>>>>> 4bf54f8 (hz)
