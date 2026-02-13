package assignment

import (
	"context"
	"radiology-assignment/internal/models"
)

type MockDataStore struct {
	GetShiftsByWorkTypeFunc           func(ctx context.Context, modality, bodyPart string, site string) ([]*models.Shift, error)
	GetRadiologistFunc                func(ctx context.Context, id string) (*models.Radiologist, error)
	GetRadiologistsFunc               func(ctx context.Context, ids []string) ([]*models.Radiologist, error)
	GetRadiologistCurrentWorkloadFunc func(ctx context.Context, radiologistID string) (int64, error)
	GetRadiologistWorkloadsFunc       func(ctx context.Context, radiologistIDs []string) (map[string]int64, error)
	GetRadiologistRVUWorkloadsFunc    func(ctx context.Context, radiologistIDs []string) (map[string]float64, error)
	SaveAssignmentFunc                func(ctx context.Context, assignment *models.Assignment) error
}

func (m *MockDataStore) GetShiftsByWorkType(ctx context.Context, modality, bodyPart string, site string) ([]*models.Shift, error) {
	return m.GetShiftsByWorkTypeFunc(ctx, modality, bodyPart, site)
}

func (m *MockDataStore) GetRadiologist(ctx context.Context, id string) (*models.Radiologist, error) {
	return m.GetRadiologistFunc(ctx, id)
}

func (m *MockDataStore) GetRadiologists(ctx context.Context, ids []string) ([]*models.Radiologist, error) {
	return m.GetRadiologistsFunc(ctx, ids)
}

func (m *MockDataStore) GetRadiologistCurrentWorkload(ctx context.Context, radiologistID string) (int64, error) {
	return m.GetRadiologistCurrentWorkloadFunc(ctx, radiologistID)
}

func (m *MockDataStore) GetRadiologistWorkloads(ctx context.Context, radiologistIDs []string) (map[string]int64, error) {
	if m.GetRadiologistWorkloadsFunc != nil {
		return m.GetRadiologistWorkloadsFunc(ctx, radiologistIDs)
	}

	// Fallback: use the single item fetcher which might be mocked by tests
	results := make(map[string]int64)
	for _, id := range radiologistIDs {
		val, err := m.GetRadiologistCurrentWorkload(ctx, id)
		if err != nil {
			return nil, err
		}
		results[id] = val
	}
	return results, nil
}

func (m *MockDataStore) GetRadiologistRVUWorkloads(ctx context.Context, radiologistIDs []string) (map[string]float64, error) {
	if m.GetRadiologistRVUWorkloadsFunc != nil {
		return m.GetRadiologistRVUWorkloadsFunc(ctx, radiologistIDs)
	}
	return make(map[string]float64), nil
}

func (m *MockDataStore) SaveAssignment(ctx context.Context, assignment *models.Assignment) error {
	return m.SaveAssignmentFunc(ctx, assignment)
}

type MockRosterService struct {
	GetByShiftFunc func(shiftID int64) []*models.RosterEntry
}

func (m *MockRosterService) GetByShift(shiftID int64) []*models.RosterEntry {
	return m.GetByShiftFunc(shiftID)
}

type MockRulesService struct {
	GetActiveFunc func() []*models.AssignmentRule
}

func (m *MockRulesService) GetActive() []*models.AssignmentRule {
	return m.GetActiveFunc()
}
