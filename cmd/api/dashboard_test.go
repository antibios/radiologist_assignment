package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"radiology-assignment/internal/models"
	"strings"
	"testing"
	"time"
)

func TestHandleDashboard(t *testing.T) {
	// Setup mock data
	assignmentsMu.Lock()
    oldAssignments := assignments
    assignments = []*models.Assignment{}
	for i := 0; i < 60; i++ {
		assignments = append(assignments, &models.Assignment{
			ID:            int64(i),
			StudyID:       fmt.Sprintf("ST%d", i),
			RadiologistID: "rad1",
			ShiftID:       1,
			AssignedAt:    time.Now(),
            Strategy:      "test",
		})
	}
	assignmentsMu.Unlock()
    defer func() {
        assignmentsMu.Lock()
        assignments = oldAssignments
        assignmentsMu.Unlock()
    }()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleDashboard)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

    body := rr.Body.String()

	// Check body for total count
    // 60 assignments
	expectedCount := "<h5>60</h5>"
    // We expect the count to be displayed in h5 tag as per template
	if !strings.Contains(body, expectedCount) {
		t.Errorf("handler returned unexpected body: missing count %v", expectedCount)
	}

    // Check for "Recent Assignments" header
    if !strings.Contains(body, "Recent Assignments") {
        t.Errorf("Body missing 'Recent Assignments' header")
    }

    // We expect ST59 (most recent) to be present
    if !strings.Contains(body, "ST59") {
        t.Errorf("Body missing most recent assignment ST59")
    }

    // We expect ST0 (oldest) to be absent because limit is 50
    // ST0 to ST9 should be absent if we show 50-59 (10 items) ... wait limit is 50.
    // 60 items. 0..59.
    // Recent should be 59 down to 10.
    // So 0 should be excluded.
    if strings.Contains(body, ">ST0<") {
         t.Errorf("Body contains oldest assignment ST0 which should be trimmed")
    }
}
