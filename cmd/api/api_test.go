package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"radiology-assignment/internal/assignment"
	"testing"
)

func TestAPI_OverAssignment(t *testing.T) {
	// Initialize engine
	engine = assignment.NewEngine(&InMemoryStore{}, &InMemoryRoster{}, &InMemoryRules{})

	// Reset global state to ensure clean test
	radiologistWorkload = make(map[string]int64)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/shifts":
			handleAPIShifts(w, r)
		case "/api/shifts/assign":
			handleAssignRadiologist(w, r)
		case "/api/simulate":
			handleSimulateAssignment(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// Custom client to not follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	shiftName := "Overload Shift"

	// 1. Create Shift via API
	log.Println("TestAPI_OverAssignment: calling CreateShift")
	resp, err := client.PostForm(ts.URL+"/api/shifts", url.Values{
		"name":        {shiftName},
		"work_type":   {"XRAY"},
		"site":        {"SiteA"},
		"credentials": {""},
		"priority":    {"0"},
	})
	if err != nil {
		t.Fatalf("Failed to create shift via API: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 303/200 for create shift, got %d", resp.StatusCode)
	}
	log.Println("TestAPI_OverAssignment: CreateShift done")

	// Find the created shift ID
	var createdShiftID int64
	shiftsMu.RLock()
	for _, s := range shifts {
		if s.Name == shiftName {
			createdShiftID = s.ID
			break
		}
	}
	shiftsMu.RUnlock()

	if createdShiftID == 0 {
		t.Fatalf("Failed to find created shift ID")
	}
	log.Printf("TestAPI_OverAssignment: Found created shift ID: %d", createdShiftID)

	// 2. Assign Radiologist via API
	log.Println("TestAPI_OverAssignment: calling Assign")
	resp, err = client.PostForm(ts.URL+"/api/shifts/assign", url.Values{
		"shift_id":        {fmt.Sprintf("%d", createdShiftID)},
		"radiologist_ids": {"rad_limited"},
	})
	if err != nil {
		t.Fatalf("Failed to assign radiologist via API: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 303/200 for assign radiologist, got %d", resp.StatusCode)
	}
	log.Println("TestAPI_OverAssignment: Assign done")

	// 3. Simulate Assignment 1 via API
	log.Println("TestAPI_OverAssignment: calling Simulate 1")
	resp1, err := http.PostForm(ts.URL+"/api/simulate", url.Values{
		"study_id": {"STUDY_1"}, // Generates Assignment ID 1
		"modality": {"XRAY"},
	})
	if err != nil {
		t.Fatalf("Failed simulation 1 API call: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp1.Body)
		t.Errorf("Expected 200 OK for first assignment, got %d. Body: %s", resp1.StatusCode, body)
	}
	log.Println("TestAPI_OverAssignment: Simulate 1 done")

	// 4. Simulate Assignment 2 via API (Should Fail)
	log.Println("TestAPI_OverAssignment: calling Simulate 2")
	resp2, err := http.PostForm(ts.URL+"/api/simulate", url.Values{
		"study_id": {"STUDY_2"}, // Generates Assignment ID 2
		"modality": {"XRAY"},
	})
	if err != nil {
		t.Fatalf("Failed simulation 2 API call: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 Service Unavailable (Over Capacity), got %d", resp2.StatusCode)
	}
	log.Println("TestAPI_OverAssignment: Simulate 2 done")
}
