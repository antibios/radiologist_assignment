package assignment

import (
	"context"
	"errors"
	"radiology-assignment/internal/models"
	"testing"
	"time"
)

func TestAssign_ShiftAndRosterResolution(t *testing.T) {
	// FR-4.1: Identify applicable shifts
	// FR-4.2: Identify radiologists from roster

	study := &models.Study{
		ID:       "study1",
		Modality: "MRI",
		BodyPart: "MSK",
		Site:     "Robina",
	}

	shift := &models.Shift{ID: 1, Name: "MRI MSK Robina", WorkType: "MRI/MSK", Sites: []string{"Robina"}}
	rad := &models.Radiologist{ID: "rad1", Status: "active", Credentials: []string{"MRI"}}
	rosterEntry := &models.RosterEntry{ShiftID: 1, RadiologistID: "rad1", Status: "active"}

	mockDB := &MockDataStore{
		GetShiftsByWorkTypeFunc: func(ctx context.Context, mod, body, site string) ([]*models.Shift, error) {
			if mod == "MRI" && body == "MSK" && site == "Robina" {
				return []*models.Shift{shift}, nil
			}
			return nil, nil
		},
		GetRadiologistFunc: func(ctx context.Context, id string) (*models.Radiologist, error) {
			if id == "rad1" {
				return rad, nil
			}
			return nil, errors.New("not found")
		},
		GetRadiologistCurrentWorkloadFunc: func(ctx context.Context, id string) (int64, error) {
			return 0, nil
		},
		SaveAssignmentFunc: func(ctx context.Context, a *models.Assignment) error {
			return nil
		},
	}

	mockRoster := &MockRosterService{
		GetByShiftFunc: func(shiftID int64) []*models.RosterEntry {
			if shiftID == 1 {
				return []*models.RosterEntry{rosterEntry}
			}
			return nil
		},
	}

	mockRules := &MockRulesService{
		GetActiveFunc: func() []*models.AssignmentRule {
			return []*models.AssignmentRule{} // No specific rules, default to primary
		},
	}

	engine := NewEngine(mockDB, mockRoster, mockRules)
	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad1" {
		t.Errorf("Expected rad1, got %s", assignment.RadiologistID)
	}
	if assignment.ShiftID != 1 {
		t.Errorf("Expected shift 1, got %d", assignment.ShiftID)
	}
}

func TestAssign_CompetencyFiltering(t *testing.T) {
	// FR-4.3: Competency-based filters

	study := &models.Study{ID: "study2", Modality: "CT", BodyPart: "HEAD", Site: "Metro"}
	shift := &models.Shift{ID: 2}

	// Rad1 has CT, Rad2 does not
	rad1 := &models.Radiologist{ID: "rad1", Status: "active", Credentials: []string{"CT", "MRI"}}
	rad2 := &models.Radiologist{ID: "rad2", Status: "active", Credentials: []string{"MRI"}}

	mockDB := &MockDataStore{
		GetShiftsByWorkTypeFunc: func(ctx context.Context, mod, body, site string) ([]*models.Shift, error) {
			return []*models.Shift{shift}, nil
		},
		GetRadiologistFunc: func(ctx context.Context, id string) (*models.Radiologist, error) {
			if id == "rad1" { return rad1, nil }
			if id == "rad2" { return rad2, nil }
			return nil, nil
		},
		GetRadiologistCurrentWorkloadFunc: func(ctx context.Context, id string) (int64, error) {
			return 0, nil
		},
		SaveAssignmentFunc: func(ctx context.Context, a *models.Assignment) error { return nil },
	}

	mockRoster := &MockRosterService{
		GetByShiftFunc: func(shiftID int64) []*models.RosterEntry {
			return []*models.RosterEntry{
				{ShiftID: 2, RadiologistID: "rad1"},
				{ShiftID: 2, RadiologistID: "rad2"},
			}
		},
	}

	mockRules := &MockRulesService{
		GetActiveFunc: func() []*models.AssignmentRule {
			// Rule to filter by competency
			return []*models.AssignmentRule{
				{
					ID: 1, PriorityOrder: 1,
					ActionType: "FILTER_COMPETENCY", // Assuming this action type implies filtering by modality
				},
			}
		},
	}

	engine := NewEngine(mockDB, mockRoster, mockRules)
	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad1" {
		t.Errorf("Expected rad1 (has CT), got %s", assignment.RadiologistID)
	}
}

func TestAssign_CapacityConstraints(t *testing.T) {
	// FR-4.6: Capacity constraints

	study := &models.Study{ID: "study3"}
	shift := &models.Shift{ID: 3}

	// Rad1 full, Rad2 available
	rad1 := &models.Radiologist{ID: "rad1", Status: "active", MaxConcurrentStudies: 2}
	rad2 := &models.Radiologist{ID: "rad2", Status: "active", MaxConcurrentStudies: 2}

	mockDB := &MockDataStore{
		GetShiftsByWorkTypeFunc: func(ctx context.Context, mod, body, site string) ([]*models.Shift, error) {
			return []*models.Shift{shift}, nil
		},
		GetRadiologistFunc: func(ctx context.Context, id string) (*models.Radiologist, error) {
			if id == "rad1" { return rad1, nil }
			if id == "rad2" { return rad2, nil }
			return nil, nil
		},
		GetRadiologistCurrentWorkloadFunc: func(ctx context.Context, id string) (int64, error) {
			if id == "rad1" { return 2, nil } // At capacity
			return 0, nil
		},
		SaveAssignmentFunc: func(ctx context.Context, a *models.Assignment) error { return nil },
	}

	mockRoster := &MockRosterService{
		GetByShiftFunc: func(shiftID int64) []*models.RosterEntry {
			return []*models.RosterEntry{
				{ShiftID: 3, RadiologistID: "rad1"},
				{ShiftID: 3, RadiologistID: "rad2"},
			}
		},
	}

	mockRules := &MockRulesService{
		GetActiveFunc: func() []*models.AssignmentRule { return []*models.AssignmentRule{} },
	}

	engine := NewEngine(mockDB, mockRoster, mockRules)
	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad2" {
		t.Errorf("Expected rad2 (available), got %s", assignment.RadiologistID)
	}
}

func TestAssign_LoadBalancing(t *testing.T) {
	// FR-4.6: Load balancing

	study := &models.Study{ID: "study4"}
	shift := &models.Shift{ID: 4}

	// Both available, Rad1 has 1 study, Rad2 has 0
	rad1 := &models.Radiologist{ID: "rad1", Status: "active", MaxConcurrentStudies: 5}
	rad2 := &models.Radiologist{ID: "rad2", Status: "active", MaxConcurrentStudies: 5}

	mockDB := &MockDataStore{
		GetShiftsByWorkTypeFunc: func(ctx context.Context, mod, body, site string) ([]*models.Shift, error) {
			return []*models.Shift{shift}, nil
		},
		GetRadiologistFunc: func(ctx context.Context, id string) (*models.Radiologist, error) {
			if id == "rad1" { return rad1, nil }
			if id == "rad2" { return rad2, nil }
			return nil, nil
		},
		GetRadiologistCurrentWorkloadFunc: func(ctx context.Context, id string) (int64, error) {
			if id == "rad1" { return 1, nil }
			return 0, nil // Less load
		},
		SaveAssignmentFunc: func(ctx context.Context, a *models.Assignment) error { return nil },
	}

	mockRoster := &MockRosterService{
		GetByShiftFunc: func(shiftID int64) []*models.RosterEntry {
			return []*models.RosterEntry{
				{ShiftID: 4, RadiologistID: "rad1"},
				{ShiftID: 4, RadiologistID: "rad2"},
			}
		},
	}

	mockRules := &MockRulesService{
		GetActiveFunc: func() []*models.AssignmentRule { return []*models.AssignmentRule{} },
	}

	engine := NewEngine(mockDB, mockRoster, mockRules)
	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad2" {
		t.Errorf("Expected rad2 (lower load), got %s", assignment.RadiologistID)
	}
}

func TestAssign_SLAEscalation(t *testing.T) {
	// FR-4.5: SLA Escalation

	// Study is old
	study := &models.Study{
		ID: "study5",
		IngestTime: time.Now().Add(-60 * time.Minute), // 60 mins old
	}
	shift := &models.Shift{ID: 5}

	rad1 := &models.Radiologist{ID: "rad1", Status: "active"}

	mockDB := &MockDataStore{
		GetShiftsByWorkTypeFunc: func(ctx context.Context, mod, body, site string) ([]*models.Shift, error) {
			return []*models.Shift{shift}, nil
		},
		GetRadiologistFunc: func(ctx context.Context, id string) (*models.Radiologist, error) {
			return rad1, nil
		},
		GetRadiologistCurrentWorkloadFunc: func(ctx context.Context, id string) (int64, error) {
			return 0, nil
		},
		SaveAssignmentFunc: func(ctx context.Context, a *models.Assignment) error { return nil },
	}

	mockRoster := &MockRosterService{
		GetByShiftFunc: func(shiftID int64) []*models.RosterEntry {
			return []*models.RosterEntry{{ShiftID: 5, RadiologistID: "rad1"}}
		},
	}

	mockRules := &MockRulesService{
		GetActiveFunc: func() []*models.AssignmentRule {
			// Rule: If > 30 mins, set escalated flag
			return []*models.AssignmentRule{
				{
					ID: 5,
					ActionType: "ESCALATE",
					ConditionFilters: map[string]interface{}{"min_age_minutes": 30},
				},
			}
		},
	}

	engine := NewEngine(mockDB, mockRoster, mockRules)
	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !assignment.Escalated {
		t.Errorf("Expected assignment to be escalated")
	}
}

func TestAssign_NoMatchingShift(t *testing.T) {
	study := &models.Study{ID: "study6"}

	mockDB := &MockDataStore{
		GetShiftsByWorkTypeFunc: func(ctx context.Context, mod, body, site string) ([]*models.Shift, error) {
			return nil, nil // No shifts
		},
	}
	mockRoster := &MockRosterService{}
	mockRules := &MockRulesService{}

	engine := NewEngine(mockDB, mockRoster, mockRules)
	_, err := engine.Assign(context.Background(), study)

	if err == nil {
		t.Fatal("Expected error when no shifts match")
	}
}

func TestAssign_NoAvailableRadiologists(t *testing.T) {
	study := &models.Study{ID: "study7"}
	shift := &models.Shift{ID: 7}

	mockDB := &MockDataStore{
		GetShiftsByWorkTypeFunc: func(ctx context.Context, mod, body, site string) ([]*models.Shift, error) {
			return []*models.Shift{shift}, nil
		},
	}
	mockRoster := &MockRosterService{
		GetByShiftFunc: func(shiftID int64) []*models.RosterEntry {
			return []*models.RosterEntry{} // Empty roster
		},
	}
	mockRules := &MockRulesService{}

	engine := NewEngine(mockDB, mockRoster, mockRules)
	_, err := engine.Assign(context.Background(), study)

	if err == nil {
		t.Fatal("Expected error when no radiologists found")
	}
}
