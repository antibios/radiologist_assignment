package assignment

import (
	"context"
	"fmt"
	"radiology-assignment/internal/models"
	"testing"
	"time"
)

// Benchmark with simulated DB latency to show N+1 impact
func BenchmarkAssignment_WithLatency(b *testing.B) {
	numRads := 100 // Smaller number to keep benchmark time reasonable
	rads := make([]*models.Radiologist, numRads)
	rosterMap := make(map[int64][]string)
	rosterList := make([]string, numRads)

	for i := 0; i < numRads; i++ {
		id := fmt.Sprintf("rad%d", i)
		rads[i] = &models.Radiologist{ID: id, Status: "active", Credentials: []string{"MRI"}}
		rosterList[i] = id
	}
	rosterMap[1] = rosterList

	shift := &models.Shift{ID: 1, WorkType: "MRI"}
	study := &models.Study{ID: "bench_study", Modality: "MRI"}

	// Custom setup to inject latency
	engine := setupEngine(b, []*models.Shift{shift}, rads, rosterMap, nil)

	// Inject latency into the mock
	mockDB := engine.db.(*MockDataStore)
	originalGet := mockDB.GetRadiologistFunc
	originalGetBulk := mockDB.GetRadiologistsFunc

	mockDB.GetRadiologistFunc = func(ctx context.Context, id string) (*models.Radiologist, error) {
		time.Sleep(100 * time.Microsecond) // Simulate 0.1ms DB latency
		return originalGet(ctx, id)
	}

	mockDB.GetRadiologistsFunc = func(ctx context.Context, ids []string) ([]*models.Radiologist, error) {
		time.Sleep(100 * time.Microsecond) // Simulate 0.1ms DB latency for bulk fetch too
		return originalGetBulk(ctx, ids)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Assign(context.Background(), study)
	}
}
