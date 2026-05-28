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
