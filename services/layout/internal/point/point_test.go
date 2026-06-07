package point

import (
	"math"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/stretchr/testify/assert"
)

func TestVecProduct(t *testing.T) {
	testCases := []struct {
		p        Point
		vector   Point
		expected float64
	}{
		{
			// Перпендикулярные (против часовой)
			p:        Point{X: 1, Y: 0},
			vector:   Point{X: 0, Y: 1},
			expected: 1,
		},
		{
			// Коллинеарны
			p:        Point{X: 1, Y: 0},
			vector:   Point{X: 2, Y: 0},
			expected: 0,
		},
		{
			// Перпендикулярные (по часовой)
			p:        Point{X: 0, Y: 1},
			vector:   Point{X: 1, Y: 0},
			expected: -1,
		},
	}

	for _, tc := range testCases {
		actual := tc.p.VecProduct(tc.vector)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestCalculatePointsDistance(t *testing.T) {
	testCases := []struct{
		p, q Point
	}{
		{
			p: Point{X: 1, Y: 0},
			q: Point{X: 4, Y: 4},
		},
		{
			p: Point{X: 1, Y: 0},
			q: Point{X: 1, Y: 0},
		},
		{
			p: Point{X: 0, Y: 1},
			q: Point{X: 0, Y: 2},
		},
	}

	for _, tc := range testCases {
		actual := CalculatePointsDistance(tc.p, tc.q)
		expected := planar.Distance(orb.Point{tc.p.X, tc.p.Y}, orb.Point{tc.q.X, tc.q.Y})

		assert.Equal(t, expected, actual)
	}
}

func TestOrientationFunction(t *testing.T) {
	testCases := []struct{
		p, q, r Point
		expected int
	}{
		{
			p: Point{X: 0, Y: 0},
			q: Point{X: 1, Y: 0},
			r: Point{X: 2, Y: 0},
			expected: 0,
		},
		{
			p: Point{X: 0, Y: 0},
			q: Point{X: 1, Y: 0},
			r: Point{X: 1, Y: -1},
			expected: 1,
		},
		{
			p: Point{X: 0, Y: 0},
			q: Point{X: 1, Y: 0},
			r: Point{X: 0, Y: 1},
			expected: 2,
		},
	}

	for _, tc := range testCases {
		actual := Orientation(tc.p, tc.q, tc.r)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestGetDirectionToPoint(t *testing.T) {
	testCases := []struct{
		p, q Point
		expected Point
	}{
		{
			p: Point{X: 0, Y: 0},
			q: Point{X: 0, Y: 3},
			expected: Point{X: 0, Y: 1},
		},
		{
			p: Point{X: 0, Y: 0},
			q: Point{X: 100, Y: 100},
			expected: Point{X: 1 / math.Sqrt(2), Y: 1 / math.Sqrt(2)},
		},
		{
			p: Point{X: 0, Y: 0},
			q: Point{X: -10, Y: 0},
			expected: Point{X: -1, Y: 0},
		},
	}

	for _, tc := range testCases {
		actual := GetDirectionToPoint(tc.p, tc.q)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestIsPointInPolygon(t *testing.T) {
	polygon := []Point{
		{X: 0, Y: 0},
		{X: 0, Y: 3},
		{X: 3, Y: 3},
		{X: 3, Y: 5},
		{X: 6, Y: 5},
		{X: 6, Y: 0},
	}

	orbPolygon := orb.Polygon{{
		orb.Point{0, 0},
		orb.Point{0, 3},
		orb.Point{3, 3},
		orb.Point{3, 5},
		orb.Point{6, 5},
		orb.Point{6, 0},
		orb.Point{0, 0},
	}}

	testCases := []struct{
		p Point
	}{
		{Point{X: 1, Y: 1}},
		{Point{X: -1, Y: 2}},
		{Point{X: 6, Y: 5}},
		{Point{X: 0, Y: 1.5}},
	}

	for _, tc := range testCases {
		actual := IsPointInPolygon(tc.p, polygon)
		expected := planar.PolygonContains(orbPolygon, orb.Point{tc.p.X, tc.p.Y})

		assert.Equal(t, expected, actual)
	}
}

func TestGetCenter(t *testing.T) {
	testCases := []struct{
		polygon []Point
		orbPolygon orb.Polygon
	}{
		{
			polygon: []Point{
				{X: 0, Y: 0},
				{X: 0, Y: 3},
				{X: 3, Y: 3},
				{X: 3, Y: 5},
				{X: 6, Y: 5},
				{X: 6, Y: 0},
			},
			orbPolygon: orb.Polygon{{
				orb.Point{0, 0},
				orb.Point{0, 3},
				orb.Point{3, 3},
				orb.Point{3, 5},
				orb.Point{6, 5},
				orb.Point{6, 0},
				orb.Point{0, 0},
			}},
		},
		{
			polygon: []Point{
				{X: 3, Y: 3},
				{X: 3, Y: 5},
				{X: 10, Y: 5},
				{X: 10, Y: 3},
			},
			orbPolygon: orb.Polygon{{
				orb.Point{3, 3},
				orb.Point{3, 5},
				orb.Point{10, 5},
				orb.Point{10, 3},
				orb.Point{3, 3},
			}},
		},
	}

	for _, tc := range testCases {
		actual := GetCenter(tc.polygon)
		expected, _ := planar.CentroidArea(tc.orbPolygon)

		if math.Abs(expected[0] - actual.X) > 1e-9 || math.Abs(expected[1] - actual.Y) > 1e-9 {
			t.Errorf("Expected: %v, actual: %v", expected, actual)
		}
	}
}

func TestGetBoundaries(t *testing.T) {
	testCases := []struct{
		polygon []Point
		orbPolygon orb.Polygon
	}{
		{
			polygon: []Point{
				{X: 0, Y: 0},
				{X: 0, Y: 3},
				{X: 3, Y: 3},
				{X: 3, Y: 5},
				{X: 6, Y: 5},
				{X: 6, Y: 0},
			},
			orbPolygon: orb.Polygon{{
				orb.Point{0, 0},
				orb.Point{0, 3},
				orb.Point{3, 3},
				orb.Point{3, 5},
				orb.Point{6, 5},
				orb.Point{6, 0},
				orb.Point{0, 0},
			}},
		},
		{
			polygon: []Point{
				{X: 3, Y: 3},
				{X: 3, Y: 5},
				{X: 10, Y: 5},
				{X: 10, Y: 3},
			},
			orbPolygon: orb.Polygon{{
				orb.Point{3, 3},
				orb.Point{3, 5},
				orb.Point{10, 5},
				orb.Point{10, 3},
				orb.Point{3, 3},
			}},
		},
	}

	for _, tc := range testCases {
		actual_min, actual_max := GetBoundaries(tc.polygon)
		expected := tc.orbPolygon.Bound()

		assert.Equal(t, expected.Min[0], actual_min.X)
		assert.Equal(t, expected.Min[1], actual_min.Y)
		assert.Equal(t, expected.Max[0], actual_max.X)
		assert.Equal(t, expected.Max[1], actual_max.Y)
	}
}

func TestGridMethodSize(t *testing.T) {
	polygon := []Point{
		{X: 0, Y: 0},
		{X: 3, Y: 0},
		{X: 3, Y: 3},
		{X: 0, Y: 3},
	}

	step := 0.5
	gridPoints, err := GenerateGridPoints(polygon, step)

	assert.NoError(t, err)
	assert.Equal(t, int((3 / step) * (3 / step)), len(gridPoints))
}

func TestIsSegmentsIntersect(t *testing.T) {
	testCases := []struct{
		seg1, seg2 Segment
		expected bool
	}{
		{
			seg1: Segment{
				From: Point{X: 1, Y: 0},
				To: Point{X: 3, Y: 0},
			},
			seg2: Segment{
				From: Point{X: 2, Y: 1},
				To: Point{X: 2, Y: -1},
			},
			expected: true,
		},
		{
			seg1: Segment{
				From: Point{X: 1, Y: 0},
				To: Point{X: 3, Y: 0},
			},
			seg2: Segment{
				From: Point{X: 1, Y: 1},
				To: Point{X: 3, Y: 1},
			},
			expected: false,
		},
		{
			seg1: Segment{
				From: Point{X: 0, Y: 0},
				To: Point{X: 3, Y: 3},
			},
			seg2: Segment{
				From: Point{X: 3, Y: 3},
				To: Point{X: 1, Y: 1},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		actual := IsSegmentsIntersect(&tc.seg1, &tc.seg2)
		
		assert.Equal(t, tc.expected, actual)
	}
}
