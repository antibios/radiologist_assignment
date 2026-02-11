package assignment

import (
	"context"
	"radiology-assignment/internal/models"
)

// DataStore defines the interface for database operations
type DataStore interface {
	GetShiftsByWorkType(ctx context.Context, modality, bodyPart string, site string) ([]*models.Shift, error)
	GetRadiologist(ctx context.Context, id string) (*models.Radiologist, error)
	GetRadiologistCurrentWorkload(ctx context.Context, radiologistID string) (int64, error)
	GetRadiologistWorkloads(ctx context.Context, radiologistIDs []string) (map[string]int64, error)
	SaveAssignment(ctx context.Context, assignment *models.Assignment) error
}

// RosterService defines the interface for roster retrieval
type RosterService interface {
	GetByShift(shiftID int64) []*models.RosterEntry
}

// RulesService defines the interface for rule retrieval
type RulesService interface {
	GetActive() []*models.AssignmentRule
}
