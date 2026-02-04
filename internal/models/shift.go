package models

import "time"

type Shift struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	WorkType            string    `json:"work_type"`
	Sites               []string  `json:"sites"`
	PriorityLevel       int       `json:"priority_level"`
	RequiredCredentials []string  `json:"required_credentials"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
