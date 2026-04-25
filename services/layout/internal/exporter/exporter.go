package exporter

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

func ExportToJSON(layout *apartment.ApartmentLayout) ([]byte, error) {
	return json.MarshalIndent(layout, "", "  ")
}
