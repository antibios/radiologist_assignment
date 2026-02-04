package models

import "time"

type Assignment struct {
	ID            int64     `json:"id"`
	StudyID       string    `json:"study_id"`
	RadiologistID string    `json:"radiologist_id"`
	ShiftID       int64     `json:"shift_id"`
	AssignedAt    time.Time `json:"assigned_at"`
	Escalated     bool      `json:"escalated"`
	Strategy      string    `json:"strategy"`
	RuleMatchedID *int64    `json:"rule_matched_id"`
	CreatedAt     time.Time `json:"created_at"`
}
