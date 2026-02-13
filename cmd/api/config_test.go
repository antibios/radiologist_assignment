package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"radiology-assignment/internal/models"
	"strings"
	"testing"
)

func TestHandleAPISites(t *testing.T) {
	// Setup
	refData = &models.ReferenceData{Sites: []models.Site{}}

	form := url.Values{}
	form.Add("code", "TEST_SITE")
	form.Add("name", "Test Site Clinic")

	req := httptest.NewRequest("POST", "/api/config/sites", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleAPISites(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect 303, got %d", w.Code)
	}

	configMu.RLock()
	if len(refData.Sites) != 1 {
		t.Errorf("Expected 1 site, got %d", len(refData.Sites))
	} else if refData.Sites[0].Code != "TEST_SITE" {
		t.Errorf("Site code mismatch: %s", refData.Sites[0].Code)
	}
	configMu.RUnlock()
}

func TestHandleDeleteSite(t *testing.T) {
	refData = &models.ReferenceData{
		Sites: []models.Site{{Code: "DEL_SITE", Name: "Delete Me"}},
	}

	form := url.Values{}
	form.Add("code", "DEL_SITE")

	req := httptest.NewRequest("POST", "/api/config/sites/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleDeleteSite(w, req)

	configMu.RLock()
	if len(refData.Sites) != 0 {
		t.Errorf("Expected 0 sites, got %d", len(refData.Sites))
	}
	configMu.RUnlock()
}

func TestHandleAPIModalities(t *testing.T) {
	refData = &models.ReferenceData{Modalities: []models.Modality{}}

	form := url.Values{}
	form.Add("code", "TEST_MOD")
	form.Add("name", "Test Modality")

	req := httptest.NewRequest("POST", "/api/config/modalities", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleAPIModalities(w, req)

	configMu.RLock()
	if len(refData.Modalities) != 1 {
		t.Errorf("Expected 1 modality, got %d", len(refData.Modalities))
	}
	configMu.RUnlock()
}

func TestHandleAPIBodyParts(t *testing.T) {
	refData = &models.ReferenceData{BodyParts: []models.BodyPart{}}

	form := url.Values{}
	form.Add("name", "TestBodyPart")

	req := httptest.NewRequest("POST", "/api/config/bodyparts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleAPIBodyParts(w, req)

	configMu.RLock()
	if len(refData.BodyParts) != 1 {
		t.Errorf("Expected 1 body part, got %d", len(refData.BodyParts))
	}
	configMu.RUnlock()
}

func TestHandleAPICredentials(t *testing.T) {
	refData = &models.ReferenceData{Credentials: []models.Credential{}}

	form := url.Values{}
	form.Add("code", "TEST_CRED")
	form.Add("name", "Test Credential")

	req := httptest.NewRequest("POST", "/api/config/credentials", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleAPICredentials(w, req)

	configMu.RLock()
	if len(refData.Credentials) != 1 {
		t.Errorf("Expected 1 credential, got %d", len(refData.Credentials))
	}
	configMu.RUnlock()
}

func TestHandleEditSite(t *testing.T) {
	refData = &models.ReferenceData{
		Sites: []models.Site{{Code: "EDIT_SITE", Name: "Edit Me"}},
	}

	form := url.Values{}
	form.Add("original_code", "EDIT_SITE")
	form.Add("code", "EDITED_SITE")
	form.Add("name", "Edited Name")

	req := httptest.NewRequest("POST", "/api/config/sites/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleEditSite(w, req)

	configMu.RLock()
	if len(refData.Sites) != 1 {
		t.Errorf("Expected 1 site, got %d", len(refData.Sites))
	} else if refData.Sites[0].Code != "EDITED_SITE" || refData.Sites[0].Name != "Edited Name" {
		t.Errorf("Site not updated correctly: %+v", refData.Sites[0])
	}
	configMu.RUnlock()
}

func TestHandleEditModality(t *testing.T) {
	refData = &models.ReferenceData{
		Modalities: []models.Modality{{Code: "EDIT_MOD", Name: "Edit Mod"}},
	}

	form := url.Values{}
	form.Add("original_code", "EDIT_MOD")
	form.Add("code", "EDITED_MOD")
	form.Add("name", "Edited Mod Name")

	req := httptest.NewRequest("POST", "/api/config/modalities/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleEditModality(w, req)

	configMu.RLock()
	if len(refData.Modalities) != 1 {
		t.Errorf("Expected 1 modality, got %d", len(refData.Modalities))
	} else if refData.Modalities[0].Code != "EDITED_MOD" || refData.Modalities[0].Name != "Edited Mod Name" {
		t.Errorf("Modality not updated correctly: %+v", refData.Modalities[0])
	}
	configMu.RUnlock()
}

func TestHandleEditBodyPart(t *testing.T) {
	refData = &models.ReferenceData{
		BodyParts: []models.BodyPart{{Name: "EditBodyPart"}},
	}

	form := url.Values{}
	form.Add("original_name", "EditBodyPart")
	form.Add("name", "EditedBodyPart")

	req := httptest.NewRequest("POST", "/api/config/bodyparts/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleEditBodyPart(w, req)

	configMu.RLock()
	if len(refData.BodyParts) != 1 {
		t.Errorf("Expected 1 body part, got %d", len(refData.BodyParts))
	} else if refData.BodyParts[0].Name != "EditedBodyPart" {
		t.Errorf("Body part not updated correctly: %+v", refData.BodyParts[0])
	}
	configMu.RUnlock()
}

func TestHandleEditCredential(t *testing.T) {
	refData = &models.ReferenceData{
		Credentials: []models.Credential{{Code: "EDIT_CRED", Name: "Edit Cred"}},
	}

	form := url.Values{}
	form.Add("original_code", "EDIT_CRED")
	form.Add("code", "EDITED_CRED")
	form.Add("name", "Edited Cred Name")

	req := httptest.NewRequest("POST", "/api/config/credentials/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleEditCredential(w, req)

	configMu.RLock()
	if len(refData.Credentials) != 1 {
		t.Errorf("Expected 1 credential, got %d", len(refData.Credentials))
	} else if refData.Credentials[0].Code != "EDITED_CRED" || refData.Credentials[0].Name != "Edited Cred Name" {
		t.Errorf("Credential not updated correctly: %+v", refData.Credentials[0])
	}
	configMu.RUnlock()
}
