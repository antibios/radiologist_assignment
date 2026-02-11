package main

import (
	"fmt"
	"radiology-assignment/internal/models"
	"testing"
	"time"
)

// BenchmarkCalendarLogic establishes a baseline for the handleCalendar grid generation logic.
// This measures the performance of pre-processing the roster and generating the calendar shifts.
func BenchmarkCalendarLogic(b *testing.B) {
	// Setup mock data representing a typical medium-to-large deployment
	numShifts := 100   // 100 different shift types/sites
	numRoster := 2000  // 2000 roster entries (roughly 1 month of coverage for 100 shifts)
	days := 30         // Standard Month View

	testShifts := make([]*models.Shift, numShifts)
	for i := 0; i < numShifts; i++ {
		testShifts[i] = &models.Shift{ID: int64(i), Name: fmt.Sprintf("Shift %d", i), WorkType: "MRI"}
	}

	testRoster := make([]*models.RosterEntry, numRoster)
	startTime := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -15)
	for i := 0; i < numRoster; i++ {
		testRoster[i] = &models.RosterEntry{
			ID:            int64(i),
			ShiftID:       int64(i % numShifts),
			RadiologistID: fmt.Sprintf("rad%d", i),
			StartDate:     startTime.AddDate(0, 0, i/numShifts),
			Status:        "active",
		}
	}

	start := startTime
	end := start.AddDate(0, 0, days)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateCalendarGrid(start, end, testShifts, testRoster)
	}
}

// generateCalendarGrid is a helper that replicates the logic in handleCalendar for benchmarking.
// It reflects the optimized implementation using map-based roster lookups.
func generateCalendarGrid(start, end time.Time, shifts []*models.Shift, roster []*models.RosterEntry) []CalendarDay {
	type rosterKey struct {
		ShiftID int64
		Year    int
		Month   time.Month
		Day     int
	}
	rosterMap := make(map[rosterKey]*models.RosterEntry)
	for _, entry := range roster {
		key := rosterKey{
			ShiftID: entry.ShiftID,
			Year:    entry.StartDate.Year(),
			Month:   entry.StartDate.Month(),
			Day:     entry.StartDate.Day(),
		}
		if _, exists := rosterMap[key]; !exists {
			rosterMap[key] = entry
		}
	}

	var days []CalendarDay
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		day := CalendarDay{Date: d}
		y, m, dayNum := d.Date()

		for _, s := range shifts {
			key := rosterKey{ShiftID: s.ID, Year: y, Month: m, Day: dayNum}
			entry, filled := rosterMap[key]
			radName := ""
			if filled {
				radName = entry.RadiologistID
			}

			cs := CalendarShift{
				ShiftName:   s.Name,
				ShiftType:   s.WorkType,
				Date:        d,
				Filled:      filled,
				Radiologist: radName,
			}
			day.Shifts = append(day.Shifts, cs)
		}
		days = append(days, day)
	}
	return days
}
