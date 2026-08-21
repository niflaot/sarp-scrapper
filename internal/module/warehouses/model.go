package warehouses

import "time"

const defaultIconURL = "https://ucp.sarp.es/assets/images/map_icons/radar_warehouse.png"

type Warehouse struct {
	ID      string   `json:"id" example:"1345"`
	Address string   `json:"address" example:"Camino del Sol, Richman, Los Santos"`
	Value   int64    `json:"value" example:"742203"`
	Icon    string   `json:"icon,omitempty"`
	ExitX   *float64 `json:"exitX,omitempty"`
	ExitY   *float64 `json:"exitY,omitempty"`
}

type Snapshot struct {
	Warehouses  []Warehouse `json:"warehouses"`
	Count       int         `json:"count" example:"1"`
	LastAttempt *time.Time  `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time  `json:"lastSuccess,omitempty"`
	LastError   string      `json:"lastError,omitempty"`
	NextRun     *time.Time  `json:"nextRun,omitempty"`
}
