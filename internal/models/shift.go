package models

import "time"

type Shift struct {
	ID                      int64     `json:"id"`
	Name                    string    `json:"name"`
	WorkType                string    `json:"work_type"`
	Sites                   []string  `json:"sites"`
	PriorityLevel           int       `json:"priority_level"`
	RequiredCredentials     []string  `json:"required_credentials"`
	PreferredRadiologistIDs []string  `json:"preferred_radiologist_ids"`
	BlockedRadiologistIDs   []string  `json:"blocked_radiologist_ids"`
	LoadBalancingStrategy   string    `json:"load_balancing_strategy"` // "count" or "rvu"
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}
