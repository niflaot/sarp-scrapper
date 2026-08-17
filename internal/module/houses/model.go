package houses

import "time"

type House struct {
	Key       string   `json:"key" example:"749" description:"In-game property key."`
	Address   string   `json:"address" example:"Olympia Avenue, Rodeo"`
	Price     int64    `json:"price" example:"1338876"`
	Available bool     `json:"available" example:"true"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

type Snapshot struct {
	Houses      []House    `json:"houses"`
	Count       int        `json:"count" example:"1"`
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
	NextRun     *time.Time `json:"nextRun,omitempty"`
}
