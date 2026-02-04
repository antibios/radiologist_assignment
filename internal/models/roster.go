package models

import "time"

type RosterEntry struct {
	ID            int64      `json:"id"`
	ShiftID       int64      `json:"shift_id"`
	RadiologistID string     `json:"radiologist_id"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
