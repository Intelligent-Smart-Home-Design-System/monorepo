package config

import "time"

type EntityConfig struct {
	ID    string        `json:"id"`
	Type  string        `json:"type"`
	Delay time.Duration `json:"delay"`
}
