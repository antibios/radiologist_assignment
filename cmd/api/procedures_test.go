package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"radiology-assignment/internal/models"
	"strings"
	"testing"
)

func TestHandleAPIProcedures(t *testing.T) {
	// Setup initial state
	procedures = []*models.Procedure{}

	// Test Create
	form := url.Values{}
	form.Add("code", "TEST1")
	form.Add("description", "Test Proc")
	form.Add("modality", "CT")
	form.Add("body_part", "HEAD")

	req := httptest.NewRequest("POST", "/api/procedures", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleAPIProcedures(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect 303, got %d", w.Code)
	}

	proceduresMu.RLock()
	if len(procedures) != 1 {
		t.Errorf("Expected 1 procedure, got %d", len(procedures))
	} else {
		p := procedures[0]
		if p.Code != "TEST1" || p.Modality != "CT" {
			t.Errorf("Procedure data mismatch: %+v", p)
		}
	}
	proceduresMu.RUnlock()
}

func TestHandleEditProcedure(t *testing.T) {
	// Setup
	procedures = []*models.Procedure{
		{Code: "TEST2", Description: "Old Desc", Modality: "MR", BodyPart: "KNEE"},
	}

	form := url.Values{}
	form.Add("code", "TEST2")
	form.Add("description", "New Desc")
	form.Add("modality", "MR")
	form.Add("body_part", "LEG")

	req := httptest.NewRequest("POST", "/api/procedures/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleEditProcedure(w, req)

	proceduresMu.RLock()
	p := procedures[0]
	if p.Description != "New Desc" || p.BodyPart != "LEG" {
		t.Errorf("Procedure not updated: %+v", p)
	}
	proceduresMu.RUnlock()
}

func TestHandleDeleteProcedure(t *testing.T) {
	// Setup
	procedures = []*models.Procedure{
		{Code: "TEST3"},
	}

	form := url.Values{}
	form.Add("code", "TEST3")

	req := httptest.NewRequest("POST", "/api/procedures/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleDeleteProcedure(w, req)

	proceduresMu.RLock()
	if len(procedures) != 0 {
		t.Errorf("Expected 0 procedures, got %d", len(procedures))
	}
	proceduresMu.RUnlock()
}

func TestRenderProcedures(t *testing.T) {
	// Setup
	procedures = []*models.Procedure{
		{Code: "P1", Description: "D1", Modality: "M1", BodyPart: "B1"},
	}

	req := httptest.NewRequest("GET", "/procedures", nil)
	w := httptest.NewRecorder()

	handleProcedures(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !contains(body, "Procedure Management") {
		t.Error("Body does not contain 'Procedure Management'")
	}
	if !contains(body, "P1") {
		t.Error("Body does not contain procedure code 'P1'")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
