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
		vec1     Point
		vec2     Point
		expected float64
	}{
		{
			vec1:     Point{X: 1, Y: 0},
			vec2:     Point{X: 0, Y: 1},
			expected: 1,
		},
		{
			vec1:     Point{X: 1, Y: 0},
			vec2:     Point{X: 2, Y: 0},
			expected: 0,
		},
		{
			vec1:     Point{X: 0, Y: 1},
			vec2:     Point{X: 1, Y: 0},
			expected: -1,
		},
	}

	for _, tc := range testCases {
		actual := tc.vec1.VecProduct(tc.vec2)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestDotProduct(t *testing.T) {
	testCases := []struct {
		vec1     Point
		vec2     Point
		expected float64
	}{
		{
			vec1:     Point{X: 1, Y: 0},
			vec2:     Point{X: 1, Y: 0},
			expected: 1,
		},
		{
			vec1:     Point{X: 1, Y: 0},
			vec2:     Point{X: 0, Y: 1},
			expected: 0,
		},
		{
			vec1:     Point{X: 1, Y: 0},
			vec2:     Point{X: -1, Y: 0},
			expected: -1,
		},
		{
			vec1:     Point{X: 3, Y: 4},
			vec2:     Point{X: 2, Y: 5},
			expected: 3*2 + 4*5,
		},
		{
			vec1:     Point{X: -2, Y: 3},
			vec2:     Point{X: 4, Y: -1},
			expected: (-2)*4 + 3*(-1),
		},
	}

	for _, tc := range testCases {
		actual := tc.vec1.DotProduct(tc.vec2)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestIsInInterval(t *testing.T) {
	testCases := []struct {
		p        Point
		p1       Point
		p2       Point
		expected bool
	}{
		{
			p:        Point{X: 3, Y: 1},
			p1:       Point{X: 1, Y: 1},
			p2:       Point{X: 5, Y: 1},
			expected: true,
		},
		{
			p:        Point{X: 3, Y: 3},
			p1:       Point{X: 1, Y: 1},
			p2:       Point{X: 5, Y: 5},
			expected: true,
		},
		{
			p:        Point{X: 6, Y: 6},
			p1:       Point{X: 1, Y: 1},
			p2:       Point{X: 5, Y: 5},
			expected: false,
		},
		{
			p:        Point{X: 0, Y: 0},
			p1:       Point{X: 0, Y: 0},
			p2:       Point{X: 0, Y: 0},
			expected: true,
		},
	}

	for _, tc := range testCases {
		actual := tc.p.IsInInterval(tc.p1.X, tc.p1.Y, tc.p2.X, tc.p2.Y)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestIsInRay(t *testing.T) {
	testCases := []struct {
		p        Point
		p1       Point
		p2       Point
		expected bool
	}{
		{
			p:        Point{X: 3, Y: 1},
			p1:       Point{X: 1, Y: 1},
			p2:       Point{X: 5, Y: 1},
			expected: true,
		},
		{
			p:        Point{X: 3, Y: 3},
			p1:       Point{X: 1, Y: 1},
			p2:       Point{X: 5, Y: 5},
			expected: true,
		},
		{
			p:        Point{X: 6, Y: 6},
			p1:       Point{X: 1, Y: 1},
			p2:       Point{X: 5, Y: 5},
			expected: true,
		},
		{
			p:        Point{X: 0, Y: 0},
			p1:       Point{X: 0, Y: 0},
			p2:       Point{X: 0, Y: 0},
			expected: true,
		},
		{
			p:        Point{X: -1, Y: 0},
			p1:       Point{X: 0, Y: 0},
			p2:       Point{X: 5, Y: 0},
			expected: false,
		},
	}

	for _, tc := range testCases {
		actual := tc.p.IsInRay(tc.p1.X, tc.p1.Y, tc.p2.X, tc.p2.Y)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestMovePointInDirection(t *testing.T) {
	testCases := []struct {
		p         Point
		direction Point
		offset    float64
		expected  Point
	}{
		{
			p:         Point{X: 0, Y: 0},
			direction: Point{X: 1, Y: 0},
			offset:    5,
			expected:  Point{X: 5, Y: 0},
		},
		{
			p:         Point{X: 10, Y: 0},
			direction: Point{X: -1, Y: 0},
			offset:    3,
			expected:  Point{X: 7, Y: 0},
		},
		{
			p:         Point{X: 0, Y: 0},
			direction: Point{X: 1, Y: 1},
			offset:    math.Sqrt2,
			expected:  Point{X: 1, Y: 1},
		},
		{
			p:         Point{X: 0, Y: 0},
			direction: Point{X: 1, Y: 1},
			offset:    5,
			expected:  Point{X: 5 / math.Sqrt2, Y: 5 / math.Sqrt2},
		},
	}

	for _, tc := range testCases {
		actual := MovePointInDirection(tc.p, tc.direction, tc.offset)

		if math.Abs(tc.expected.X-actual.X) > epsilon || math.Abs(tc.expected.Y-actual.Y) > epsilon {
			t.Errorf("Expected: %v, actual: %v", tc.expected, actual)
		}
	}
}

func TestCalculatePointsDistance(t *testing.T) {
	testCases := []struct {
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
	testCases := []struct {
		p, q, r  Point
		expected int
	}{
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: 1, Y: 0},
			r:        Point{X: 2, Y: 0},
			expected: 0,
		},
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: 1, Y: 0},
			r:        Point{X: 1, Y: -1},
			expected: 1,
		},
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: 1, Y: 0},
			r:        Point{X: 0, Y: 1},
			expected: 2,
		},
	}

	for _, tc := range testCases {
		actual := Orientation(tc.p, tc.q, tc.r)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestGetDirectionToPoint(t *testing.T) {
	testCases := []struct {
		p, q     Point
		expected Point
	}{
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: 0, Y: 3},
			expected: Point{X: 0, Y: 1},
		},
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: 100, Y: 100},
			expected: Point{X: 1 / math.Sqrt(2), Y: 1 / math.Sqrt(2)},
		},
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: -10, Y: 0},
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

	testCases := []struct {
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
	testCases := []struct {
		polygon    []Point
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

		if math.Abs(expected[0]-actual.X) > epsilon || math.Abs(expected[1]-actual.Y) > epsilon {
			t.Errorf("Expected: %v, actual: %v", expected, actual)
		}
	}
}

func TestGetBoundaries(t *testing.T) {
	testCases := []struct {
		polygon    []Point
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
	assert.Equal(t, int((3/step)*(3/step)), len(gridPoints))
}

func TestIsSegmentsIntersect(t *testing.T) {
	testCases := []struct {
		seg1, seg2 Segment
		expected   bool
	}{
		{
			seg1: Segment{
				From: Point{X: 1, Y: 0},
				To:   Point{X: 3, Y: 0},
			},
			seg2: Segment{
				From: Point{X: 2, Y: 1},
				To:   Point{X: 2, Y: -1},
			},
			expected: true,
		},
		{
			seg1: Segment{
				From: Point{X: 1, Y: 0},
				To:   Point{X: 3, Y: 0},
			},
			seg2: Segment{
				From: Point{X: 1, Y: 1},
				To:   Point{X: 3, Y: 1},
			},
			expected: false,
		},
		{
			seg1: Segment{
				From: Point{X: 0, Y: 0},
				To:   Point{X: 3, Y: 3},
			},
			seg2: Segment{
				From: Point{X: 3, Y: 3},
				To:   Point{X: 1, Y: 1},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		actual := IsSegmentsIntersect(&tc.seg1, &tc.seg2)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestIntervalLength(t *testing.T) {
	testCases := []struct {
		interval Interval
		expected float64
	}{
		{
			interval: Interval{
				From: 2,
				To:   10,
			},
			expected: 8,
		},
		{
			interval: Interval{
				From: -2,
				To:   2,
			},
			expected: 4,
		},
		{
			interval: Interval{
				From: 0,
				To:   0.5,
			},
			expected: 0.5,
		},
	}

	for _, tc := range testCases {
		actual := tc.interval.Length()

		assert.Equal(t, tc.expected, actual)
	}
}

func TestNormalize(t *testing.T) {
	testCases := []struct {
		vec      Point
		expected Point
	}{
		{
			vec:      Point{X: 3, Y: 4},
			expected: Point{X: 0.6, Y: 0.8},
		},
		{
			vec:      Point{X: 1, Y: 0},
			expected: Point{X: 1, Y: 0},
		},
		{
			vec:      Point{X: -5, Y: 12},
			expected: Point{X: -5.0 / 13, Y: 12.0 / 13},
		},
	}

	for _, tc := range testCases {
		actual := Normalize(tc.vec)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestSegmentLength(t *testing.T) {
	testCases := []struct {
		seg      Segment
		expected float64
	}{
		{
			seg: Segment{
				From: Point{X: 0, Y: 0},
				To:   Point{X: 1, Y: 0},
			},
			expected: 1,
		},
		{
			seg: Segment{
				From: Point{X: 0, Y: 0},
				To:   Point{X: 0, Y: 0},
			},
			expected: 0,
		},
		{
			seg: Segment{
				From: Point{X: 100, Y: 5},
				To:   Point{X: 53, Y: 5},
			},
			expected: 47,
		},
	}

	for _, tc := range testCases {
		actual := tc.seg.Length()

		assert.Equal(t, tc.expected, actual)
	}
}

func TestNormalVectors(t *testing.T) {
	testCases := []struct {
		seg           Segment
		expectedLeft  Point
		expectedRight Point
	}{
		{
			seg: Segment{
				From: Point{X: 0, Y: 0},
				To:   Point{X: 2, Y: 0},
			},
			expectedLeft:  Point{X: 0, Y: 2},
			expectedRight: Point{X: 0, Y: -2},
		},
		{
			seg: Segment{
				From: Point{X: 0, Y: 0},
				To:   Point{X: 0, Y: 3},
			},
			expectedLeft:  Point{X: -3, Y: 0},
			expectedRight: Point{X: 3, Y: 0},
		},
		{
			seg: Segment{
				From: Point{X: 1, Y: 1},
				To:   Point{X: 4, Y: 4},
			},
			expectedLeft:  Point{X: -3, Y: 3},
			expectedRight: Point{X: 3, Y: -3},
		},
	}

	for _, tc := range testCases {
		actualLeft, actualRight := tc.seg.NormalVectors()

		assert.Equal(t, tc.expectedLeft, *actualLeft)
		assert.Equal(t, tc.expectedRight, *actualRight)
	}
}

func TestIsPointOnSegment(t *testing.T) {
	testCases := []struct {
		p, q, r  Point
		expected bool
	}{
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: 0, Y: 1},
			r:        Point{X: 0, Y: 2},
			expected: true,
		},
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: 5, Y: 0},
			r:        Point{X: 10, Y: 0},
			expected: true,
		},
		{
			p:        Point{X: 5, Y: 0},
			q:        Point{X: 5, Y: 7},
			r:        Point{X: 5, Y: 10},
			expected: true,
		},
		{
			p:        Point{X: 0, Y: 0},
			q:        Point{X: 20, Y: 7},
			r:        Point{X: -1, Y: -1},
			expected: false,
		},
		{
			p:        Point{X: 5, Y: 0},
			q:        Point{X: 10, Y: 0},
			r:        Point{X: 9, Y: 0},
			expected: false,
		},
	}

	for _, tc := range testCases {
		actual := IsPointOnSegment(tc.p, tc.q, tc.r)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestClosestPointOnSegment(t *testing.T) {
	testCases := []struct {
		point    Point
		seg      Segment
		expected Point
	}{
		{
			point: Point{X: 3, Y: 2},
			seg: Segment{
				From: Point{X: 0, Y: 0},
				To:   Point{X: 10, Y: 0},
			},
			expected: Point{X: 3, Y: 0},
		},
		{
			point: Point{X: 3, Y: 7},
			seg: Segment{
				From: Point{X: 3, Y: 0},
				To:   Point{X: 3, Y: 10},
			},
			expected: Point{X: 3, Y: 7},
		},
		{
			point: Point{X: 0, Y: 0},
			seg: Segment{
				From: Point{X: 2, Y: -2},
				To:   Point{X: 2, Y: 100},
			},
			expected: Point{X: 2, Y: 0},
		},
		{
			point: Point{X: 4, Y: 6},
			seg: Segment{
				From: Point{X: 0, Y: 0},
				To:   Point{X: 10, Y: 10},
			},
			expected: Point{X: 5, Y: 5},
		},
	}

	for _, tc := range testCases {
		actual := ClosestPointOnSegment(tc.point, tc.seg)

		if math.Abs(tc.expected.X-actual.X) > epsilon || math.Abs(tc.expected.Y-actual.Y) > epsilon {
			t.Errorf("Expected: %v, actual: %v", tc.expected, actual)
		}
	}
}
