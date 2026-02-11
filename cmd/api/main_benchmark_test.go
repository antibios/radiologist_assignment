package main

import (
	"context"
	"fmt"
	"radiology-assignment/internal/models"
	"testing"
)

func BenchmarkGetRadiologist(b *testing.B) {
	// Setup: Create a large number of radiologists
	numRadiologists := 10000
	originalRadiologists := radiologists
	originalRadiologistsMap := radiologistsMap

	// New slice
	newRadiologists := make([]*models.Radiologist, numRadiologists)
	newRadiologistsMap := make(map[string]*models.Radiologist)

	for i := 0; i < numRadiologists; i++ {
		rad := &models.Radiologist{
			ID:                   fmt.Sprintf("rad%d", i),
			FirstName:            "Test",
			LastName:             "User",
			MaxConcurrentStudies: 5,
		}
		newRadiologists[i] = rad
		newRadiologistsMap[rad.ID] = rad
	}

	radiologists = newRadiologists
	radiologistsMap = newRadiologistsMap

	// Ensure we restore original state after benchmark
	defer func() {
		radiologists = originalRadiologists
		radiologistsMap = originalRadiologistsMap
	}()

	store := &InMemoryStore{}
	ctx := context.Background()
	targetID := fmt.Sprintf("rad%d", numRadiologists-1) // Worst case: last element

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetRadiologist(ctx, targetID)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
