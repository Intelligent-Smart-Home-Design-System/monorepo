package climate

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/geometry"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const airConditionerCasesPath = "testdata/air_conditioner_free_segments.json"

type airConditionerCasesFile struct {
	Note            string               `json:"note,omitempty"`
	CoordinateOrder string               `json:"coordinate_order,omitempty"`
	Cases           []airConditionerCase `json:"cases"`
}

type airConditionerCase struct {
	ID                    int           `json:"id"`
	Screenshot            string        `json:"screenshot,omitempty"`
	RoomVerticesClockwise []jsonPoint   `json:"room_vertices_clockwise"`
	RoomSegments          []jsonSegment `json:"room_segments"`
	Beds                  []jsonBed     `json:"beds"`
	GreenSegments         []jsonSegment `json:"green_segments"`
	BlackHatchingRegion   *jsonRect     `json:"black_hatching_region,omitempty"`
	ActualGreenSegments   []jsonSegment `json:"actual_green_segments,omitempty"`
}

type jsonBed struct {
	ID                string      `json:"id"`
	VerticesClockwise []jsonPoint `json:"vertices_clockwise"`
	Rect              *jsonRect   `json:"rect,omitempty"`
}

type jsonRect struct {
	XMin float64 `json:"x_min"`
	YMin float64 `json:"y_min"`
	XMax float64 `json:"x_max"`
	YMax float64 `json:"y_max"`
}

type jsonPoint [2]float64
type jsonSegment [2]jsonPoint

func TestFindOkWindWallIntervalsFromCases(t *testing.T) {
	casesFile := readAirConditionerCases(t)
	actualCasesFile := airConditionerCasesFile{
		Note:            casesFile.Note,
		CoordinateOrder: casesFile.CoordinateOrder,
		Cases:           make([]airConditionerCase, 0, len(casesFile.Cases)),
	}

	for _, tc := range casesFile.Cases {
		tc := tc
		t.Run(fmt.Sprintf("case_%d", tc.ID), func(t *testing.T) {
			ap := buildApartmentFromCase(tc)
			zonedAp := apartment.Build(ap)
			require.Len(t, zonedAp.ZonedRooms, 1)

			room := zonedAp.ZonedRooms[0].OrigRoom
			noWindZones := collectNoWindZones(zonedAp.ZonedRooms[0].GetFurniture())
			actualSegments := normalizeSegments(intervalsToJSONSegments(t, ap.Walls, FindOkWindWallIntervals(room, noWindZones, 0)))
			expectedSegments := normalizeSegments(tc.GreenSegments)

			caseWithActual := tc
			caseWithActual.ActualGreenSegments = actualSegments
			actualCasesFile.Cases = append(actualCasesFile.Cases, caseWithActual)

			assert.Equal(t, expectedSegments, actualSegments)
		})
	}

	writeActualAirConditionerCases(t, actualCasesFile)
}

func readAirConditionerCases(t *testing.T) airConditionerCasesFile {
	t.Helper()

	data, err := os.ReadFile(airConditionerCasesPath)
	require.NoError(t, err)

	var casesFile airConditionerCasesFile
	require.NoError(t, json.Unmarshal(data, &casesFile))
	require.NotEmpty(t, casesFile.Cases)

	return casesFile
}

func buildApartmentFromCase(tc airConditionerCase) *apartment.Apartment {
	walls := make([]apartment.Wall, 0, len(tc.RoomSegments))
	wallIDs := make([]string, 0, len(tc.RoomSegments))
	for i, segment := range tc.RoomSegments {
		wallID := fmt.Sprintf("case_%d_wall_%d", tc.ID, i+1)
		wallIDs = append(wallIDs, wallID)
		walls = append(walls, apartment.Wall{
			ID:     wallID,
			Points: []point.Point{segment[0].toPoint(), segment[1].toPoint()},
		})
	}

	furnitureIDs := make([]string, 0, len(tc.Beds))
	furniture := make([]apartment.Furniture, 0, len(tc.Beds))
	for _, bed := range tc.Beds {
		furnitureIDs = append(furnitureIDs, bed.ID)
		furniture = append(furniture, apartment.Furniture{
			ID:     bed.ID,
			Name:   apartment.FirnitureBed,
			Points: jsonPointsToPoints(bed.VerticesClockwise),
			Rooms:  []string{fmt.Sprintf("case_%d_room", tc.ID)},
		})
	}

	roomID := fmt.Sprintf("case_%d_room", tc.ID)
	return &apartment.Apartment{
		Walls: walls,
		Rooms: []apartment.Room{
			{
				ID:        roomID,
				Name:      apartment.RoomBedroom,
				Area:      jsonPointsToPoints(tc.RoomVerticesClockwise),
				Walls:     wallIDs,
				Furniture: furnitureIDs,
			},
		},
		Furniture: furniture,
	}
}

func intervalsToJSONSegments(t *testing.T, walls []apartment.Wall, intervals map[string][]point.Interval) []jsonSegment {
	t.Helper()

	wallsByID := make(map[string]apartment.Wall, len(walls))
	for _, w := range walls {
		wallsByID[w.ID] = w
	}

	segments := make([]jsonSegment, 0)
	for wallID, wallIntervals := range intervals {
		w, ok := wallsByID[wallID]
		if !ok {
			t.Fatalf("wall %q not found", wallID)
		}

		wallSegment := point.Segment{From: w.Points[0], To: w.Points[1]}
		for _, interval := range wallIntervals {
			segmentPoints := geometry.IntervalToPoints(wallSegment, &interval)
			require.Len(t, segmentPoints, 2)
			segments = append(segments, jsonSegment{pointToJSON(segmentPoints[0]), pointToJSON(segmentPoints[1])})
		}
	}

	return segments
}

func normalizeSegments(segments []jsonSegment) []jsonSegment {
	result := make([]jsonSegment, 0, len(segments))
	for _, segment := range segments {
		normalized := jsonSegment{normalizeJSONPoint(segment[0]), normalizeJSONPoint(segment[1])}
		if lessJSONPoint(normalized[1], normalized[0]) {
			normalized[0], normalized[1] = normalized[1], normalized[0]
		}
		result = append(result, normalized)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i][0] != result[j][0] {
			return lessJSONPoint(result[i][0], result[j][0])
		}
		return lessJSONPoint(result[i][1], result[j][1])
	})

	return result
}

func normalizeJSONPoint(p jsonPoint) jsonPoint {
	return jsonPoint{roundCoord(p[0]), roundCoord(p[1])}
}

func lessJSONPoint(a, b jsonPoint) bool {
	if a[0] != b[0] {
		return a[0] < b[0]
	}
	return a[1] < b[1]
}

func jsonPointsToPoints(points []jsonPoint) []point.Point {
	result := make([]point.Point, 0, len(points))
	for _, p := range points {
		result = append(result, p.toPoint())
	}
	return result
}

func (p jsonPoint) toPoint() point.Point {
	return point.Point{X: p[0], Y: p[1]}
}

func pointToJSON(p point.Point) jsonPoint {
	return jsonPoint{roundCoord(p.X), roundCoord(p.Y)}
}

func roundCoord(v float64) float64 {
	if math.Abs(v) < 1e-9 {
		return 0
	}
	return math.Round(v*1e9) / 1e9
}

func writeActualAirConditionerCases(t *testing.T, casesFile airConditionerCasesFile) {
	t.Helper()

	data, err := json.MarshalIndent(casesFile, "", "  ")
	require.NoError(t, err)

	// The full actual result is always available in the verbose test output;
	// set WRITE_AIR_CONDITIONER_ACTUAL=1
	// when a stable file artifact is needed for debugging or fixture updates.
	if os.Getenv("WRITE_AIR_CONDITIONER_ACTUAL") == "1" {
		outputPath := filepath.Join("testdata", "air_conditioner_actual_free_segments.json")
		require.NoError(t, os.WriteFile(outputPath, data, 0o600))
		t.Logf("actual air conditioner free segments written to %s", outputPath)
	}

	t.Logf("actual air conditioner free segments:\n%s", data)
}
