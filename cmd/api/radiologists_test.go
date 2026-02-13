package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"radiology-assignment/internal/assignment"
	"radiology-assignment/internal/models"
	"strings"
	"testing"
)

func TestRadiologistHandlers(t *testing.T) {
	// Initialize global variables for testing
	engine = assignment.NewEngine(&InMemoryStore{}, &InMemoryRoster{}, &InMemoryRules{})
	radiologists = []*models.Radiologist{}
	radiologistsMap = make(map[string]*models.Radiologist)

	t.Run("CreateRadiologist", func(t *testing.T) {
		form := url.Values{}
		form.Add("id", "test_rad")
		form.Add("first_name", "Test")
		form.Add("last_name", "Radiologist")
		form.Add("max_concurrent_studies", "3")
		form.Add("credentials", "CT")
		form.Add("credentials", "MRI")
		form.Add("specialties", "Neuro")

		req, err := http.NewRequest("POST", "/api/config/radiologists", strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleAPIRadiologists)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusSeeOther {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusSeeOther)
		}

		// Verify radiologist was added
		radiologistsMu.RLock()
		defer radiologistsMu.RUnlock()
		if len(radiologists) != 1 {
			t.Errorf("expected 1 radiologist, got %d", len(radiologists))
		}
		rad := radiologists[0]
		if rad.ID != "test_rad" {
			t.Errorf("expected ID test_rad, got %s", rad.ID)
		}
		if len(rad.Credentials) != 2 {
			t.Errorf("expected 2 credentials, got %d", len(rad.Credentials))
		}
	})

	t.Run("EditRadiologist", func(t *testing.T) {
		form := url.Values{}
		form.Add("id", "test_rad")
		form.Add("first_name", "Updated")
		form.Add("last_name", "Name")
		form.Add("max_concurrent_studies", "5")
		form.Add("status", "inactive")
		form.Add("credentials", "US")

		req, err := http.NewRequest("POST", "/api/config/radiologists/edit", strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleEditRadiologist)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusSeeOther {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusSeeOther)
		}

		// Verify update
		radiologistsMu.RLock()
		defer radiologistsMu.RUnlock()
		rad := radiologists[0]
		if rad.FirstName != "Updated" {
			t.Errorf("expected first name Updated, got %s", rad.FirstName)
		}
		if rad.Status != "inactive" {
			t.Errorf("expected status inactive, got %s", rad.Status)
		}
		if len(rad.Credentials) != 1 || rad.Credentials[0] != "US" {
			t.Errorf("expected credentials [US], got %v", rad.Credentials)
		}
	})

	t.Run("DeleteRadiologist", func(t *testing.T) {
		form := url.Values{}
		form.Add("id", "test_rad")

		req, err := http.NewRequest("POST", "/api/config/radiologists/delete", strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleDeleteRadiologist)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusSeeOther {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusSeeOther)
		}

		// Verify deletion
		radiologistsMu.RLock()
		defer radiologistsMu.RUnlock()
		if len(radiologists) != 0 {
			t.Errorf("expected 0 radiologists, got %d", len(radiologists))
		}
	})
}
