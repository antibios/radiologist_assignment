package models

import "time"

type Radiologist struct {
	ID                  string    `json:"id"`
	FirstName           string    `json:"first_name"`
	LastName            string    `json:"last_name"`
	Credentials         []string  `json:"credentials"`
	Specialties         []string  `json:"specialties"`
	MaxConcurrentStudies int       `json:"max_concurrent_studies"`
	Status              string    `json:"status"` // active, inactive
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
