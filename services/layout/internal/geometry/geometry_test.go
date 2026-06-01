package geometry

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/stretchr/testify/assert"
)

func TestIsSegmentIntersectPolygon(t *testing.T) {
	polygon := []point.Point{
		{X: 0, Y: 0},
		{X: 0, Y: 5},
		{X: 3, Y: 5},
		{X: 3, Y: 2},
		{X: 7, Y: 2},
		{X: 7, Y: 0},
	}

	testCases := []struct{
		polygon []point.Point
		seg point.Segment
		expected bool
	}{
		{
			polygon: polygon,
			seg: point.Segment{
				From: point.Point{X: -1, Y: 1},
				To: point.Point{X: 100, Y: 1},
			},
			expected: true,
		},
		{
			polygon: polygon,
			seg: point.Segment{
				From: point.Point{X: 0, Y: 0},
				To: point.Point{X: 100, Y: 0},
			},
			expected: true,
		},
		{
			polygon: polygon,
			seg: point.Segment{
				From: point.Point{X: 1, Y: -1},
				To: point.Point{X: 100, Y: -1},
			},
			expected: false,
		},
		{
			polygon: polygon,
			seg: point.Segment{
				From: point.Point{X: -5, Y: 0},
				To: point.Point{X: 0, Y: 5},
			},
			expected: true,
		},
		{
			polygon: polygon,
			seg: point.Segment{
				From: point.Point{X: 1, Y: 1},
				To: point.Point{X: 2, Y: 1},
			},
			expected: true,
		},
		{
			polygon: polygon,
			seg: point.Segment{
				From: point.Point{X: -7, Y: 0},
				To: point.Point{X: -1, Y: 0.5},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		actual := IsSegmentIntersectPolygon(tc.polygon, tc.seg)

		assert.Equal(t, tc.expected, actual)
	}
} 

func TestFreeIntervals(t *testing.T) {
	t.Run("стена без блокировок", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		intervals := tracker.FreeIntervals(2)

		assert.Equal(t, 1, len(intervals))
		assert.Equal(t, 0.0, intervals[0].From)
		assert.Equal(t, 20.0, intervals[0].To)
	})

	t.Run("один заблокированный интервал посередине", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		tracker.Block(point.Interval{From: 7.5, To: 13.5})
		intervals := tracker.FreeIntervals(2)

		assert.Equal(t, 2, len(intervals))
		assert.Equal(t, 0.0, intervals[0].From)
		assert.Equal(t, 7.5, intervals[0].To)
		assert.Equal(t, 13.5, intervals[1].From)
		assert.Equal(t, 20.0, intervals[1].To)
	})

	t.Run("один заблокированный интервал вначале", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		tracker.Block(point.Interval{From: 0.0, To: 13.5})
		intervals := tracker.FreeIntervals(5)

		assert.Equal(t, 1, len(intervals))
		assert.Equal(t, 13.5, intervals[0].From)
		assert.Equal(t, 20.0, intervals[0].To)
	})

	t.Run("проверка минимальной длины", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		tracker.Block(point.Interval{From: 2.0, To: 13.5})
		intervals := tracker.FreeIntervals(5)

		assert.Equal(t, 1, len(intervals))
		assert.Equal(t, 13.5, intervals[0].From)
		assert.Equal(t, 20.0, intervals[0].To)
	})

	t.Run("два перекрывающихся заблокированных интервала", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		tracker.Block(point.Interval{From: 2.0, To: 13.5})
		tracker.Block(point.Interval{From: 15, To: 17})
		intervals := tracker.FreeIntervals(1)

		assert.Equal(t, 3, len(intervals))
		assert.Equal(t, 0.0, intervals[0].From)
		assert.Equal(t, 2.0, intervals[0].To)
		assert.Equal(t, 13.5, intervals[1].From)
		assert.Equal(t, 15.0, intervals[1].To)
		assert.Equal(t, 17.0, intervals[2].From)
		assert.Equal(t, 20.0, intervals[2].To)
	})

	t.Run("защищенная область всередине заблокированной", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		tracker.Block(point.Interval{From: 2.0, To: 13.5})
		tracker.Protect(point.Interval{From: 10, To: 12})
		intervals := tracker.FreeIntervals(1)

		assert.Equal(t, 3, len(intervals))
		assert.Equal(t, 0.0, intervals[0].From)
		assert.Equal(t, 2.0, intervals[0].To)
		assert.Equal(t, 10.0, intervals[1].From)
		assert.Equal(t, 12.0, intervals[1].To)
		assert.Equal(t, 13.5, intervals[2].From)
		assert.Equal(t, 20.0, intervals[2].To)
	})

	t.Run("защищенная область пересекает заблокированную", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		tracker.Block(point.Interval{From: 2.0, To: 10})
		tracker.Protect(point.Interval{From: -2, To: 5})
		intervals := tracker.FreeIntervals(1)

		assert.Equal(t, 2, len(intervals))
		assert.Equal(t, 0.0, intervals[0].From)
		assert.Equal(t, 5.0, intervals[0].To)
		assert.Equal(t, 10.0, intervals[1].From)
		assert.Equal(t, 20.0, intervals[1].To)
	})

	t.Run("защищенная область перекрывает заблокированную", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		tracker.Block(point.Interval{From: 2.0, To: 10})
		tracker.Protect(point.Interval{From: -2, To: 5})
		tracker.Protect(point.Interval{From: 3, To: 10})
		intervals := tracker.FreeIntervals(1)

		assert.Equal(t, 1, len(intervals))
		assert.Equal(t, 0.0, intervals[0].From)
		assert.Equal(t, 20.0, intervals[0].To)
	})

	t.Run("защищенная область не пересекает заблокированную", func(t *testing.T) {
		tracker := NewWallIntervalTracker(20.0)
		tracker.Block(point.Interval{From: 2.0, To: 10})
		tracker.Protect(point.Interval{From: 15, To: 17})
		intervals := tracker.FreeIntervals(2.5)

		assert.Equal(t, 1, len(intervals))
		assert.Equal(t, 10.0, intervals[0].From)
		assert.Equal(t, 20.0, intervals[0].To)
	})
}
