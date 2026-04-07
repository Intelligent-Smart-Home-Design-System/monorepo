package domain

import (
	"net/http"
	"time"
)

type ScrapeTask struct {
	Source   string
	PageType string
	URL      string
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
