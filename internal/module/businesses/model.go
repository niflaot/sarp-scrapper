package businesses

import "time"

type Business struct {
	ID    string `json:"id" example:"17"`
	Name  string `json:"name" example:"24/7 Market"`
	Value int64  `json:"value" example:"250000"`
	Icon  string `json:"icon,omitempty"`
}

type Snapshot struct {
	Businesses  []Business `json:"businesses"`
	Count       int        `json:"count" example:"1"`
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
	NextRun     *time.Time `json:"nextRun,omitempty"`
}
