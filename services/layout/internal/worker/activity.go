package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/activity"
	"golang.org/x/sync/semaphore"
)

type ActivityService struct {
	settings Settings
	metrics  *Collector
	logger   zerolog.Logger
	sem      *semaphore.Weighted
}

type layoutArtifact struct {
	RequestID      string              `json:"request_id"`
	PlacementCount int                 `json:"placement_count"`
	MinPrice       int                 `json:"min_price"`
	MaxPrice       int                 `json:"max_price"`
	Placements     []placementArtifact `json:"placements"`
}

type placementArtifact struct {
	RoomID      string        `json:"room_id"`
	DeviceID    string        `json:"device_id"`
	DeviceType  string        `json:"device_type"`
	DeviceTrack string        `json:"device_track"`
	Point       pointArtifact `json:"point"`
}

type pointArtifact struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func NewActivityService(settings Settings, metrics *Collector, logger zerolog.Logger) *ActivityService {
	return &ActivityService{
		settings: settings,
		metrics:  metrics,
		logger:   logger,
		sem:      semaphore.NewWeighted(settings.ComputeConcurrency),
	}
}

func (s *ActivityService) BuildLayout(ctx context.Context, input LayoutActivityInput) (*LayoutActivityOutput, error) {
	start := time.Now()
	s.metrics.concurrentRuns.Inc()
	defer s.metrics.concurrentRuns.Dec()

	if err := s.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquire worker semaphore: %w", err)
	}
	defer s.sem.Release(1)

	info := activity.GetInfo(ctx)
	activityLogger := s.logger.With().
		Str("request_id", input.RequestID).
		Str("workflow_id", info.WorkflowExecution.ID).
		Str("workflow_run_id", info.WorkflowExecution.RunID).
		Str("activity_id", info.ActivityID).
		Str("activity_type", info.ActivityType.Name).
		Str("task_queue", info.TaskQueue).
		Logger()

	ctx, span := otel.Tracer(s.settings.ServiceName).Start(ctx, "layout.build_layout")
	defer span.End()
	activityLogger = traceLogger(ctx, activityLogger)

	activityLogger.Info().Str("apartment_path", input.ApartmentPath).Str("output_path", input.OutputPath).Msg("Layout activity started")
	activity.RecordHeartbeat(ctx, "started")

	apartmentStruct, err := loadApartment(input.ApartmentPath)
	if err != nil {
		s.metrics.record("failure", time.Since(start))
		return nil, err
	}
	prepareApartment(apartmentStruct)

	tracksConfig, err := configs.LoadTracksConfig(s.settings.TracksConfigPath)
	if err != nil {
		s.metrics.record("failure", time.Since(start))
		return nil, fmt.Errorf("load tracks config: %w", err)
	}

	devicesConfig, err := configs.LoadDevicesConfig(s.settings.DevicesConfigPath)
	if err != nil {
		s.metrics.record("failure", time.Since(start))
		return nil, fmt.Errorf("load devices config: %w", err)
	}

	rulesStorage := storage.NewStorage()
	loadSecurityRules(rulesStorage, devicesConfig)

	layoutEngine := engine.NewEngine(rulesStorage, tracksConfig, devicesConfig)
	layoutResult, err := layoutEngine.PlaceDevices(apartmentStruct, input.SelectedLevels)
	if err != nil {
		s.metrics.record("failure", time.Since(start))
		return nil, fmt.Errorf("place devices: %w", err)
	}

	priceInfo := layoutEngine.CalculateLayoutPrice(layoutResult)
	if err := writeArtifact(input.RequestID, input.OutputPath, layoutResult, priceInfo); err != nil {
		s.metrics.record("failure", time.Since(start))
		return nil, err
	}

	placements := countPlacements(layoutResult)
	output := &LayoutActivityOutput{
		RequestID:      input.RequestID,
		OutputPath:     input.OutputPath,
		PlacementCount: placements,
		MinPrice:       priceInfo.MinPrice,
		MaxPrice:       priceInfo.MaxPrice,
	}

	duration := time.Since(start)
	s.metrics.record("success", duration)
	activity.RecordHeartbeat(ctx, "completed")
	activityLogger.Info().
		Int("placement_count", placements).
		Int("min_price", priceInfo.MinPrice).
		Int("max_price", priceInfo.MaxPrice).
		Dur("duration", duration).
		Msg("Layout activity completed")

	return output, nil
}

func loadApartment(path string) (*apartment.Apartment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read apartment input: %w", err)
	}

	var model apartment.Apartment
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("decode apartment input: %w", err)
	}
	return &model, nil
}

func writeArtifact(requestID string, outputPath string, layoutResult any, priceInfo *engine.PriceInfo) error {
	artifact := layoutArtifact{
		RequestID:      requestID,
		PlacementCount: countPlacements(layoutResult),
		MinPrice:       priceInfo.MinPrice,
		MaxPrice:       priceInfo.MaxPrice,
		Placements:     flattenPlacements(layoutResult),
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}

	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return fmt.Errorf("encode layout artifact: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write layout artifact: %w", err)
	}
	return nil
}

func prepareApartment(apartmentStruct *apartment.Apartment) {
	switch typed := any(apartmentStruct).(type) {
	case interface{ MakeDependency() }:
		typed.MakeDependency()
	case interface{ Index() }:
		typed.Index()
	}
}

func loadSecurityRules(rulesStorage *storage.Storage, devicesConfig *configs.Devices) {
	switch typed := any(rulesStorage).(type) {
	case interface{ LoadAllSecurityRules(*configs.Devices) }:
		typed.LoadAllSecurityRules(devicesConfig)
	case interface{ LoadAllSecurityRules() }:
		typed.LoadAllSecurityRules()
	}
}

func countPlacements(layoutResult any) int {
	if layoutResult == nil {
		return 0
	}

	placementsValue := reflect.ValueOf(layoutResult)
	if placementsValue.Kind() == reflect.Pointer {
		if placementsValue.IsNil() {
			return 0
		}
		placementsValue = placementsValue.Elem()
	}

	placementsField := placementsValue.FieldByName("Placements")
	if !placementsField.IsValid() || placementsField.Kind() != reflect.Map {
		return 0
	}

	total := 0
	for _, roomKey := range placementsField.MapKeys() {
		roomPlacements := placementsField.MapIndex(roomKey)
		switch roomPlacements.Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:
			total += roomPlacements.Len()
		}
	}
	return total
}

func flattenPlacements(layoutResult any) []placementArtifact {
	if layoutResult == nil {
		return nil
	}

	placementsValue := reflect.ValueOf(layoutResult)
	if placementsValue.Kind() == reflect.Pointer {
		if placementsValue.IsNil() {
			return nil
		}
		placementsValue = placementsValue.Elem()
	}

	placementsField := placementsValue.FieldByName("Placements")
	if !placementsField.IsValid() || placementsField.Kind() != reflect.Map {
		return nil
	}

	placements := make([]placementArtifact, 0, countPlacements(layoutResult))
	for _, roomKey := range placementsField.MapKeys() {
		roomID := roomKey.String()
		roomPlacements := placementsField.MapIndex(roomKey)

		switch roomPlacements.Kind() {
		case reflect.Map:
			for _, placementKey := range roomPlacements.MapKeys() {
				if artifact, ok := placementArtifactFromValue(roomID, roomPlacements.MapIndex(placementKey)); ok {
					placements = append(placements, artifact)
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < roomPlacements.Len(); i++ {
				if artifact, ok := placementArtifactFromValue(roomID, roomPlacements.Index(i)); ok {
					placements = append(placements, artifact)
				}
			}
		}
	}
	return placements
}

func placementArtifactFromValue(roomID string, placementValue reflect.Value) (placementArtifact, bool) {
	if placementValue.Kind() == reflect.Pointer {
		if placementValue.IsNil() {
			return placementArtifact{}, false
		}
		placementValue = placementValue.Elem()
	}

	deviceField := placementValue.FieldByName("Device")
	if !deviceField.IsValid() {
		return placementArtifact{}, false
	}
	if deviceField.Kind() == reflect.Pointer {
		if deviceField.IsNil() {
			return placementArtifact{}, false
		}
		deviceField = deviceField.Elem()
	}

	pointField := placementValue.FieldByName("Place")
	if !pointField.IsValid() {
		pointField = placementValue.FieldByName("Position")
	}
	if !pointField.IsValid() {
		return placementArtifact{}, false
	}
	if pointField.Kind() == reflect.Pointer {
		if pointField.IsNil() {
			return placementArtifact{}, false
		}
		pointField = pointField.Elem()
	}

	deviceTrack := stringField(deviceField, "DeviceTrack")
	if deviceTrack == "" {
		deviceTrack = stringField(deviceField, "Track")
	}

	return placementArtifact{
		RoomID:      roomID,
		DeviceID:    stringField(deviceField, "ID"),
		DeviceType:  stringField(deviceField, "Type"),
		DeviceTrack: deviceTrack,
		Point: pointArtifact{
			X: floatField(pointField, "X"),
			Y: floatField(pointField, "Y"),
		},
	}, true
}

func stringField(value reflect.Value, fieldName string) string {
	field := value.FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

func floatField(value reflect.Value, fieldName string) float64 {
	field := value.FieldByName(fieldName)
	if !field.IsValid() {
		return 0
	}

	switch field.Kind() {
	case reflect.Float32, reflect.Float64:
		return field.Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(field.Uint())
	default:
		return 0
	}
}
