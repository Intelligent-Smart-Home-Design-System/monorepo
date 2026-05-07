package domain

type ExtractedListing struct {
	Id                 int
	Brand              string
	Model              *string
	Category           string
	CategoryConfidence float32
	DeviceAttributes   map[string]any
	LLM                string
	TaxonomyVersion    string
}

type ScrapedDirectCompatibility struct {
	Brand     string
	Model     string
	Ecosystem string
	Protocol  string
}

type Device struct {
	Id                  int
	Brand               string
	Model               *string
	Category            string
	DeviceAttributes    map[string]any
	TaxonomyVersion     string
	Listings            []*ExtractedListing
	DirectCompatibility []*DirectCompatibility
	BridgeCompatibility []*BridgeCompatibility
}

type DirectCompatibility struct {
	Ecosystem string
	Protocol  string
}

type BridgeCompatibility struct {
	SourceEcosystem string
	TargetEcosystem string
	Protocol        string
}

type Catalog struct {
	Devices []*Device
}
