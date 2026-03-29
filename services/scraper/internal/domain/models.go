package domain

import (
	"net/http"
	"time"
)

type ScrapeTask struct {
	ID       int
	Source   string
	PageType string
	URL      string
}

type ScrapeResult struct {
	Err       error
	TrackedPageID int
    DurationMs    int
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
