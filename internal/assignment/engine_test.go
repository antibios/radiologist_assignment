package assignment

import (
	"context"
	"errors"
	"radiology-assignment/internal/models"
	"testing"
	"time"
)

// Helper for boilerplate setup
func setupEngine(t testing.TB, shifts []*models.Shift, radiologists []*models.Radiologist, roster map[int64][]string, rules []*models.AssignmentRule) *Engine {
	// Optimize lookup for benchmark
	radMap := make(map[string]*models.Radiologist)
	for _, r := range radiologists {
		radMap[r.ID] = r
	}

	mockDB := &MockDataStore{
		GetShiftsByWorkTypeFunc: func(ctx context.Context, mod, body, site string) ([]*models.Shift, error) {
			return shifts, nil
		},
		GetRadiologistFunc: func(ctx context.Context, id string) (*models.Radiologist, error) {
			if r, ok := radMap[id]; ok {
				return r, nil
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
			var entries []*models.RosterEntry
			if rads, ok := roster[shiftID]; ok {
				for _, rID := range rads {
					entries = append(entries, &models.RosterEntry{ShiftID: shiftID, RadiologistID: rID})
				}
			}
			return entries
		},
	}

	mockRules := &MockRulesService{
		GetActiveFunc: func() []*models.AssignmentRule {
			return rules
		},
	}

	return NewEngine(mockDB, mockRoster, mockRules)
}

func TestAssign_ShiftAndRosterResolution(t *testing.T) {
	study := &models.Study{ID: "study1", Modality: "MRI", BodyPart: "MSK", Site: "Robina"}
	shift := &models.Shift{ID: 1, Name: "MRI MSK Robina", WorkType: "MRI/MSK", Sites: []string{"Robina"}}
	rad := &models.Radiologist{ID: "rad1", Status: "active", Credentials: []string{"MRI"}}

	engine := setupEngine(t, []*models.Shift{shift}, []*models.Radiologist{rad}, map[int64][]string{1: {"rad1"}}, nil)

	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad1" {
		t.Errorf("Expected rad1, got %s", assignment.RadiologistID)
	}
}

func TestAssign_CompetencyFiltering(t *testing.T) {
	study := &models.Study{ID: "study2", Modality: "CT"}
	shift := &models.Shift{ID: 2}
	rad1 := &models.Radiologist{ID: "rad1", Status: "active", Credentials: []string{"CT"}}
	rad2 := &models.Radiologist{ID: "rad2", Status: "active", Credentials: []string{"MRI"}}

	rules := []*models.AssignmentRule{{ID: 1, ActionType: "FILTER_COMPETENCY", PriorityOrder: 1}}

	engine := setupEngine(t, []*models.Shift{shift}, []*models.Radiologist{rad1, rad2}, map[int64][]string{2: {"rad1", "rad2"}}, rules)

	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad1" {
		t.Errorf("Expected rad1, got %s", assignment.RadiologistID)
	}
}

func TestAssign_CapacityConstraints(t *testing.T) {
	study := &models.Study{ID: "study3"}
	shift := &models.Shift{ID: 3}

	rad1 := &models.Radiologist{ID: "rad1", Status: "active", MaxConcurrentStudies: 2}
	rad2 := &models.Radiologist{ID: "rad2", Status: "active", MaxConcurrentStudies: 2}

	engine := setupEngine(t, []*models.Shift{shift}, []*models.Radiologist{rad1, rad2}, map[int64][]string{3: {"rad1", "rad2"}}, nil)

	// Override mockDB for this test to simulate load
	engine.db.(*MockDataStore).GetRadiologistCurrentWorkloadFunc = func(ctx context.Context, id string) (int64, error) {
		if id == "rad1" {
			return 2, nil
		}
		return 0, nil
	}

	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad2" {
		t.Errorf("Expected rad2, got %s", assignment.RadiologistID)
	}
}

func TestAssign_LoadBalancing(t *testing.T) {
	study := &models.Study{ID: "study4"}
	shift := &models.Shift{ID: 4}

	rad1 := &models.Radiologist{ID: "rad1", Status: "active", MaxConcurrentStudies: 5}
	rad2 := &models.Radiologist{ID: "rad2", Status: "active", MaxConcurrentStudies: 5}

	engine := setupEngine(t, []*models.Shift{shift}, []*models.Radiologist{rad1, rad2}, map[int64][]string{4: {"rad1", "rad2"}}, nil)

	engine.db.(*MockDataStore).GetRadiologistCurrentWorkloadFunc = func(ctx context.Context, id string) (int64, error) {
		if id == "rad1" {
			return 1, nil
		}
		return 0, nil
	}

	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad2" {
		t.Errorf("Expected rad2, got %s", assignment.RadiologistID)
	}
}

func TestAssign_SLAEscalation(t *testing.T) {
	study := &models.Study{ID: "study5", IngestTime: time.Now().Add(-60 * time.Minute)}
	shift := &models.Shift{ID: 5}
	rad1 := &models.Radiologist{ID: "rad1", Status: "active"}

	rules := []*models.AssignmentRule{{
		ID: 5, ActionType: "ESCALATE",
		ConditionFilters: map[string]interface{}{"min_age_minutes": 30},
	}}

	engine := setupEngine(t, []*models.Shift{shift}, []*models.Radiologist{rad1}, map[int64][]string{5: {"rad1"}}, rules)

	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !assignment.Escalated {
		t.Errorf("Expected assignment to be escalated")
	}
}

func TestAssign_SpecialArrangement(t *testing.T) {
	study := &models.Study{ID: "study_vip", Urgency: "STAT"}
	shift := &models.Shift{ID: 6}

	rad1 := &models.Radiologist{ID: "rad1", Status: "active"}    // Regular
	rad2 := &models.Radiologist{ID: "rad_vip", Status: "active"} // VIP

	// Rule: If STAT, assign to rad_vip
	rules := []*models.AssignmentRule{{
		ID: 6, ActionType: "ASSIGN_TO_RADIOLOGIST", ActionTarget: "rad_vip",
		PriorityOrder:    1,
		ConditionFilters: map[string]interface{}{"urgency": "STAT"},
	}}

	engine := setupEngine(t, []*models.Shift{shift}, []*models.Radiologist{rad1, rad2}, map[int64][]string{6: {"rad1", "rad_vip"}}, rules)

	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if assignment.RadiologistID != "rad_vip" {
		t.Errorf("Expected rad_vip, got %s", assignment.RadiologistID)
	}
}

func TestAssign_Overflow(t *testing.T) {
	// Scenario: Primary shift (High Prio) is full. Overflow shift (Low Prio) is available.
	// Currently this is handled by load balancing across all matching shifts.

	study := &models.Study{ID: "study_overflow"}
	primaryShift := &models.Shift{ID: 10, PriorityLevel: 10}
	overflowShift := &models.Shift{ID: 11, PriorityLevel: 1}

	rad1 := &models.Radiologist{ID: "rad1", Status: "active", MaxConcurrentStudies: 1} // Primary
	rad2 := &models.Radiologist{ID: "rad2", Status: "active", MaxConcurrentStudies: 1} // Overflow

	engine := setupEngine(t, []*models.Shift{primaryShift, overflowShift}, []*models.Radiologist{rad1, rad2},
		map[int64][]string{10: {"rad1"}, 11: {"rad2"}}, nil)

	engine.db.(*MockDataStore).GetRadiologistCurrentWorkloadFunc = func(ctx context.Context, id string) (int64, error) {
		if id == "rad1" {
			return 1, nil
		} // Full
		return 0, nil // Available
	}

	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// Should pick rad2 because rad1 is at capacity
	if assignment.RadiologistID != "rad2" {
		t.Errorf("Expected overflow to rad2, got %s", assignment.RadiologistID)
	}
}

func TestAssign_TieredEscalation(t *testing.T) {
	// Scenario: Study is 20 mins old.
	// Rule 1: > 15 mins -> Soft Alert (mocked as log or just no-op in logic, but rule matches)
	// Rule 2: > 30 mins -> Escalate

	study := &models.Study{ID: "study_tiered", IngestTime: time.Now().Add(-20 * time.Minute)}
	shift := &models.Shift{ID: 8}
	rad1 := &models.Radiologist{ID: "rad1", Status: "active"}

	rules := []*models.AssignmentRule{
		{ID: 81, ActionType: "SOFT_ALERT", PriorityOrder: 1, ConditionFilters: map[string]interface{}{"min_age_minutes": 15}},
		{ID: 82, ActionType: "ESCALATE", PriorityOrder: 2, ConditionFilters: map[string]interface{}{"min_age_minutes": 30}},
	}

	engine := setupEngine(t, []*models.Shift{shift}, []*models.Radiologist{rad1}, map[int64][]string{8: {"rad1"}}, rules)

	assignment, err := engine.Assign(context.Background(), study)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// Should match Rule 81, but not 82. So Escalated flag should be false.
	// But we don't capture "Soft Alert" in assignment struct in current model,
	// assuming it triggers side effect (notification).
	// But we can check that Escalated is FALSE.

	if assignment.Escalated {
		t.Errorf("Expected NOT escalated (only soft alert)")
	}

	// Now make it 40 mins old
	study.IngestTime = time.Now().Add(-40 * time.Minute)
	assignment, _ = engine.Assign(context.Background(), study)

	if !assignment.Escalated {
		t.Errorf("Expected escalated (hard reassignment)")
	}
}

func TestAssign_NoMatchingShift(t *testing.T) {
	study := &models.Study{ID: "study6"}
	engine := setupEngine(t, nil, nil, nil, nil) // No shifts
	_, err := engine.Assign(context.Background(), study)
	if err == nil {
		t.Fatal("Expected error when no shifts match")
	}
}

func TestAssign_NoAvailableRadiologists(t *testing.T) {
	study := &models.Study{ID: "study7"}
	shift := &models.Shift{ID: 7}
	engine := setupEngine(t, []*models.Shift{shift}, nil, nil, nil) // Empty roster
	_, err := engine.Assign(context.Background(), study)
	if err == nil {
		t.Fatal("Expected error when no radiologists found")
	}
}
