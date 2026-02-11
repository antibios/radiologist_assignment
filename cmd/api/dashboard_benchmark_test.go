package main

import (
	"fmt"
	"radiology-assignment/internal/models"
	"testing"
	"time"
)

// Helper to create assignments without touching global state
func createBenchmarkAssignments(n int) []*models.Assignment {
	a := make([]*models.Assignment, n)
	for i := 0; i < n; i++ {
		a[i] = &models.Assignment{
			ID:            int64(i),
			StudyID:       fmt.Sprintf("ST%d", i),
			RadiologistID: "rad1",
			ShiftID:       1,
			AssignedAt:    time.Now(),
		}
	}
    return a
}

func BenchmarkDashboardOriginal(b *testing.B) {
    // Setup a large slice to simulate load
    assignments := createBenchmarkAssignments(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
        // Simulate the inefficient block
		recent := make([]*models.Assignment, len(assignments))
		copy(recent, assignments)
		// Reverse order for display
		for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
			recent[i], recent[j] = recent[j], recent[i]
		}
        // Access len for AssignmentsCount
        _ = len(recent)
	}
}

func BenchmarkDashboardOptimized(b *testing.B) {
    // Setup a large slice to simulate load
    assignments := createBenchmarkAssignments(10000)
    limit := 50

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
        // Simulate the optimized block
        totalCount := len(assignments)

        l := limit
        if l > totalCount {
            l = totalCount
        }

        start := totalCount - l
        // Make slice of size 'l'
		recent := make([]*models.Assignment, l)
        // Copy only the last 'l' elements
		copy(recent, assignments[start:])

		// Reverse order for display
		for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
			recent[i], recent[j] = recent[j], recent[i]
		}

        // We still need totalCount for AssignmentsCount
        _ = totalCount
        _ = len(recent) // Should be 'limit'
	}
}
