package main

import (
	"context"
	"radiology-assignment/internal/models"
	"testing"
	"time"
)

func BenchmarkGetRadiologistCurrentWorkload(b *testing.B) {
	// Setup
	store := &InMemoryStore{}
	ctx := context.Background()
	radID := "rad1"

	// Populate assignments with a mix of target radiologist and others
	assignmentsMu.Lock()
	assignments = []*models.Assignment{}
	// Reset the map as well since we are resetting assignments
	radiologistWorkload = make(map[string]int64)

	// Add 10,000 assignments for the target radiologist
	for i := 0; i < 10000; i++ {
		a := &models.Assignment{
			ID:            int64(i),
			RadiologistID: radID,
			AssignedAt:    time.Now(),
		}
		assignments = append(assignments, a)
		radiologistWorkload[a.RadiologistID]++
	}

	// Add 5,000 assignments for other radiologists
	for i := 0; i < 5000; i++ {
		a := &models.Assignment{
			ID:            int64(10000 + i),
			RadiologistID: "other_rad",
			AssignedAt:    time.Now(),
		}
		assignments = append(assignments, a)
		radiologistWorkload[a.RadiologistID]++
	}
	assignmentsMu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetRadiologistCurrentWorkload(ctx, radID)
	}
}
