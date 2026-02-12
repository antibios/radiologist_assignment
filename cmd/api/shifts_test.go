package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"radiology-assignment/internal/models"
)

func TestHandleAPIShifts(t *testing.T) {
	// Setup
	shifts = []*models.Shift{}

	form := url.Values{}
	form.Add("name", "Test Shift")
	form.Add("work_type", "CT")
	form.Add("priority", "1")
	form.Add("sites", "SiteA")
	form.Add("sites", "SiteB")
	form.Add("credentials", "CT_Cert")
	form.Add("credentials", "MD")

	req := httptest.NewRequest("POST", "/api/shifts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleAPIShifts(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect 303, got %d", w.Code)
	}

	shiftsMu.RLock()
	if len(shifts) != 1 {
		t.Errorf("Expected 1 shift, got %d", len(shifts))
	} else {
		s := shifts[0]
		if s.Name != "Test Shift" {
			t.Errorf("Name mismatch: %s", s.Name)
		}
		if len(s.Sites) != 2 {
			t.Errorf("Expected 2 sites, got %d", len(s.Sites))
		}
		if len(s.RequiredCredentials) != 2 {
			t.Errorf("Expected 2 credentials, got %d", len(s.RequiredCredentials))
		}
	}
	shiftsMu.RUnlock()
}
