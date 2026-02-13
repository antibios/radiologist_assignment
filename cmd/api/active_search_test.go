package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleActiveSearch_Radiologist(t *testing.T) {
	// Construct datastar signal JSON
	signals := map[string]string{"radiologistSearch": "John"}
	signalsJSON, _ := json.Marshal(signals)
	query := url.Values{}
	query.Set("type", "radiologist")
	query.Set("datastar", string(signalsJSON))

	req, err := http.NewRequest("GET", "/active_search?"+query.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleActiveSearch)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "John Doe") {
		t.Errorf("handler returned unexpected body: does not contain 'John Doe'. Body: %s", body)
	}
}

func TestHandleActiveSearch_Site(t *testing.T) {
	signals := map[string]string{"siteSearch": "SiteA"}
	signalsJSON, _ := json.Marshal(signals)
	query := url.Values{}
	query.Set("type", "site")
	query.Set("datastar", string(signalsJSON))

	req, err := http.NewRequest("GET", "/active_search?"+query.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleActiveSearch)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Site A Clinic") {
		t.Errorf("handler returned unexpected body: does not contain 'Site A Clinic'")
	}
}

func TestHandleActiveSearch_Procedure(t *testing.T) {
	signals := map[string]string{"procedureSearch": "CTHEAD"}
	signalsJSON, _ := json.Marshal(signals)
	query := url.Values{}
	query.Set("type", "procedure")
	query.Set("datastar", string(signalsJSON))

	req, err := http.NewRequest("GET", "/active_search?"+query.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleActiveSearch)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "CT Head without contrast") {
		t.Errorf("handler returned unexpected body: does not contain 'CT Head without contrast'")
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   int
	}{
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"a", "a", 0},
		{"", "abc", 3},
	}

	for _, tt := range tests {
		if got := Levenshtein(tt.s1, tt.s2); got != tt.want {
			t.Errorf("Levenshtein(%q, %q) = %v, want %v", tt.s1, tt.s2, got, tt.want)
		}
	}
}
