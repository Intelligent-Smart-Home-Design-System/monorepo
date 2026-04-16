package domain

import (
	"net/http"
	"time"
)

type ExtractedListing struct {
	Id               int
	Brand            string
	Model            *string
	Category         string
	DeviceAttributes map[string]any
}

type ScrapeResult struct {
	Err       error
	Resources []Resource
}

// assuming GET method
type Resource struct {
	Name           string
	URL            string
	RequestHeaders http.Header

	StatusCode      int
	Status          string
	ResponseHeaders http.Header
	ResponseBody    []byte

	Timestamp time.Time // when it was fetched
}
