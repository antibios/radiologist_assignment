package assignment

import (
	"context"
	"fmt"
	"radiology-assignment/internal/models"
	"testing"
)

// BenchStore implements DataStore with the inefficient O(M) workload lookup
type BenchStore struct {
	assignments []*models.Assignment
	shifts      []*models.Shift
	rads        map[string]*models.Radiologist
}

func (s *BenchStore) GetShiftsByWorkType(ctx context.Context, modality, bodyPart, site string) ([]*models.Shift, error) {
	return s.shifts, nil
}

func (s *BenchStore) GetRadiologist(ctx context.Context, id string) (*models.Radiologist, error) {
	if r, ok := s.rads[id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("not found")
}

func (s *BenchStore) GetRadiologists(ctx context.Context, ids []string) ([]*models.Radiologist, error) {
	var results []*models.Radiologist
	for _, id := range ids {
		if r, ok := s.rads[id]; ok {
			results = append(results, r)
		}
	}
	return results, nil
}

// Inefficient implementation: O(M) where M is total assignments
func (s *BenchStore) GetRadiologistCurrentWorkload(ctx context.Context, radiologistID string) (int64, error) {
	count := int64(0)
	for _, a := range s.assignments {
		if a.RadiologistID == radiologistID {
			count++
		}
	}
	return count, nil
}

// Efficient implementation: O(M) for all requested IDs
func (s *BenchStore) GetRadiologistWorkloads(ctx context.Context, radiologistIDs []string) (map[string]int64, error) {
	counts := make(map[string]int64)
	targetIDs := make(map[string]bool)
	for _, id := range radiologistIDs {
		counts[id] = 0
		targetIDs[id] = true
	}

	for _, a := range s.assignments {
		if targetIDs[a.RadiologistID] {
			counts[a.RadiologistID]++
		}
	}
	return counts, nil
}

func (s *BenchStore) SaveAssignment(ctx context.Context, assignment *models.Assignment) error {
	s.assignments = append(s.assignments, assignment)
	return nil
}

// Ensure BenchStore implements DataStore
var _ DataStore = &BenchStore{}

func BenchmarkFilterByCapacity_NPlusOne(b *testing.B) {
	// Setup: 1000 candidates, 10,000 existing assignments
	numRads := 1000
	numAssignments := 10000

	assignments := make([]*models.Assignment, numAssignments)
	rads := make(map[string]*models.Radiologist)
	rosterEntries := make([]*models.RosterEntry, 0, numRads)

	// Create radiologists and roster
	for i := 0; i < numRads; i++ {
		id := fmt.Sprintf("rad%d", i)
		rads[id] = &models.Radiologist{
			ID:                   id,
			Status:               "active",
			MaxConcurrentStudies: 5,
		}
		rosterEntries = append(rosterEntries, &models.RosterEntry{
			RadiologistID: id,
			ShiftID:       1,
		})
	}

	// Create random assignments
	for i := 0; i < numAssignments; i++ {
		radID := fmt.Sprintf("rad%d", i%numRads)
		assignments[i] = &models.Assignment{
			ID:            int64(i),
			RadiologistID: radID,
		}
	}

	store := &BenchStore{
		assignments: assignments,
		shifts:      []*models.Shift{{ID: 1, Name: "Shift 1"}},
		rads:        rads,
	}

	roster := &MockRosterService{
		GetByShiftFunc: func(shiftID int64) []*models.RosterEntry {
			return rosterEntries
		},
	}

	rules := &MockRulesService{
		GetActiveFunc: func() []*models.AssignmentRule {
			return []*models.AssignmentRule{}
		},
	}

	engine := NewEngine(store, roster, rules)

	// Create candidates manually to test filterByCapacity directly if possible,
	// or just run Assign.
	// Since filterByCapacity is private, we can use a trick:
	// Use reflect or just copy the logic? No, we want to test the actual code.
	// Since we are in package assignment, we CAN access private methods of Engine!

	candidates := make([]*candidate, numRads)
	for i := 0; i < numRads; i++ {
		id := fmt.Sprintf("rad%d", i)
		candidates[i] = &candidate{
			Radiologist: rads[id],
			ShiftID:     1,
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.filterByCapacity(ctx, candidates)
		if err != nil {
			b.Fatalf("Error: %v", err)
		}
	}
}
