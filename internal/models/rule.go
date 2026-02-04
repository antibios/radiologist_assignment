package models

import "time"

type AssignmentRule struct {
	ID               int64                  `json:"id"`
	Name             string                 `json:"name"`
	PriorityOrder    int                    `json:"priority_order"`
	ConditionFilters map[string]interface{} `json:"condition_filters"`
	ActionType       string                 `json:"action_type"` // ASSIGN_TO_SHIFT, ASSIGN_TO_RADIOLOGIST, ESCALATE
	ActionTarget     string                 `json:"action_target"`
	Enabled          bool                   `json:"enabled"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// Matches checks if the rule applies to the given study
func (r *AssignmentRule) Matches(study *Study) bool {
	// Implementation will go here or in engine logic
	// For now, we keep the struct definition
	return false
}
