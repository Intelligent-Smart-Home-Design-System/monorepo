package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	apartmentStruct.MakeDependency()

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
	rulesStorage.LoadAllSecurityRules()

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

func writeArtifact(requestID string, outputPath string, layoutResult *apartment.ApartmentLayout, priceInfo *engine.PriceInfo) error {
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

func countPlacements(layoutResult *apartment.ApartmentLayout) int {
	if layoutResult == nil {
		return 0
	}

	total := 0
	for _, roomPlacements := range layoutResult.Placements {
		total += len(roomPlacements)
	}
	return total
}

func flattenPlacements(layoutResult *apartment.ApartmentLayout) []placementArtifact {
	if layoutResult == nil {
		return nil
	}

	placements := make([]placementArtifact, 0, countPlacements(layoutResult))
	for roomID, roomPlacements := range layoutResult.Placements {
		for _, placement := range roomPlacements {
			placements = append(placements, placementArtifact{
				RoomID:      roomID,
				DeviceID:    placement.Device.ID,
				DeviceType:  placement.Device.Type,
				DeviceTrack: placement.Device.DeviceTrack,
				Point: pointArtifact{
					X: placement.Place.X,
					Y: placement.Place.Y,
				},
			})
		}
	}
	return placements
}
