package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"radiology-assignment/internal/models"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// Mock Data Store
	rulesMu sync.RWMutex
	rules   = []*models.AssignmentRule{
		{ID: 1, Name: "Priority Stroke", PriorityOrder: 1, ActionType: "ESCALATE", Enabled: true},
		{ID: 2, Name: "MSK Load Balance", PriorityOrder: 2, ActionType: "ASSIGN_TO_SHIFT", Enabled: true},
	}

	assignments = []*models.Assignment{
		{ID: 101, StudyID: "ST1001", RadiologistID: "rad1", ShiftID: 1, Strategy: "load_balanced", AssignedAt: time.Now().Add(-5 * time.Minute)},
		{ID: 102, StudyID: "ST1002", RadiologistID: "rad2", ShiftID: 1, Strategy: "primary", AssignedAt: time.Now().Add(-2 * time.Minute)},
		{ID: 103, StudyID: "ST1003", RadiologistID: "rad_vip", ShiftID: 2, Strategy: "special_arrangement", AssignedAt: time.Now().Add(-1 * time.Minute)},
	}

	shiftsMu sync.RWMutex
	shifts   = []*models.Shift{
		{ID: 1, Name: "Morning MRI", WorkType: "MRI", Sites: []string{"SiteA"}, PriorityLevel: 1, RequiredCredentials: []string{"MRI"}},
		{ID: 2, Name: "Night CT", WorkType: "CT", Sites: []string{"SiteB"}, PriorityLevel: 2, RequiredCredentials: []string{"CT"}},
	}

	rosterMu sync.RWMutex
	roster   = []*models.RosterEntry{
		{ID: 1, ShiftID: 1, RadiologistID: "rad1", StartDate: time.Now(), Status: "active"},
	}

	// Mock Radiologists for assignment
	radiologists = []*models.Radiologist{
		{ID: "rad1", FirstName: "John", LastName: "Doe"},
		{ID: "rad2", FirstName: "Jane", LastName: "Smith"},
		{ID: "rad3", FirstName: "Bob", LastName: "Jones"},
	}
)

type DashboardData struct {
	AssignmentsCount int
	ActiveRads       int
	PendingStudies   int
	RecentAssignments []*models.Assignment
}

type RulesData struct {
	Rules []*models.AssignmentRule
}

type ShiftsData struct {
	Shifts       []*models.Shift
	Roster       map[int64][]*models.RosterEntry
	Radiologists []*models.Radiologist
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))))
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/rules", handleRules)
	http.HandleFunc("/api/rules", handleAPIRules)
	http.HandleFunc("/api/rules/edit", handleEditRule)
	http.HandleFunc("/api/rules/delete", handleDeleteRule)

	http.HandleFunc("/shifts", handleShifts)
	http.HandleFunc("/api/shifts", handleAPIShifts)
	http.HandleFunc("/api/shifts/edit", handleEditShift)
	http.HandleFunc("/api/shifts/delete", handleDeleteShift)
	http.HandleFunc("/api/shifts/assign", handleAssignRadiologist)

	log.Printf("API/UI Server started on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func render(w http.ResponseWriter, tmplName string, data interface{}, files ...string) {
	// Include layout in all renders
	allFiles := append([]string{"ui/templates/layout.html"}, files...)
	tmpl, err := template.ParseFiles(allFiles...)
	if err != nil {
		http.Error(w, "Template Parse Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Template Execute Error: "+err.Error(), http.StatusInternalServerError)
	}
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := DashboardData{
		AssignmentsCount: 142,
		ActiveRads:       18,
		PendingStudies:   3,
		RecentAssignments: assignments,
	}
	render(w, "dashboard", data, "ui/templates/dashboard.html")
}

func handleRules(w http.ResponseWriter, r *http.Request) {
	rulesMu.RLock()
	data := RulesData{
		Rules: rules,
	}
	rulesMu.RUnlock()

	render(w, "rules", data, "ui/templates/rules.html")
}

func handleAPIRules(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		action := r.FormValue("action")

		rulesMu.Lock()
		newRule := &models.AssignmentRule{
			ID: int64(len(rules) + 1),
			Name: name,
			ActionType: action,
			Enabled: true,
			PriorityOrder: len(rules) + 1,
		}
		rules = append(rules, newRule)
		rulesMu.Unlock()

		http.Redirect(w, r, "/rules", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleEditRule(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		idStr := r.FormValue("id")
		name := r.FormValue("name")
		action := r.FormValue("action")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		rulesMu.Lock()
		for _, rule := range rules {
			if rule.ID == id {
				rule.Name = name
				rule.ActionType = action
				break
			}
		}
		rulesMu.Unlock()

		http.Redirect(w, r, "/rules", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		idStr := r.FormValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		rulesMu.Lock()
		newRules := []*models.AssignmentRule{}
		for _, rule := range rules {
			if rule.ID != id {
				newRules = append(newRules, rule)
			}
		}
		rules = newRules
		rulesMu.Unlock()

		http.Redirect(w, r, "/rules", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Shifts Handlers

func handleShifts(w http.ResponseWriter, r *http.Request) {
	shiftsMu.RLock()
	rosterMu.RLock()

	// Map roster to shift IDs
	rosterMap := make(map[int64][]*models.RosterEntry)
	for _, entry := range roster {
		rosterMap[entry.ShiftID] = append(rosterMap[entry.ShiftID], entry)
	}

	data := ShiftsData{
		Shifts:       shifts,
		Roster:       rosterMap,
		Radiologists: radiologists,
	}

	rosterMu.RUnlock()
	shiftsMu.RUnlock()

	render(w, "shifts", data, "ui/templates/shifts.html")
}

func handleAPIShifts(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		workType := r.FormValue("work_type")
		site := r.FormValue("site")
		priorityStr := r.FormValue("priority")
		creds := r.FormValue("credentials")

		priority, _ := strconv.Atoi(priorityStr)

		shiftsMu.Lock()
		newShift := &models.Shift{
			ID: int64(len(shifts) + 1),
			Name: name,
			WorkType: workType,
			Sites: []string{site},
			PriorityLevel: priority,
			RequiredCredentials: strings.Split(creds, ","),
			CreatedAt: time.Now(),
		}
		shifts = append(shifts, newShift)
		shiftsMu.Unlock()

		http.Redirect(w, r, "/shifts", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleEditShift(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		idStr := r.FormValue("id")
		name := r.FormValue("name")

		id, _ := strconv.ParseInt(idStr, 10, 64)

		shiftsMu.Lock()
		for _, s := range shifts {
			if s.ID == id {
				s.Name = name
				break
			}
		}
		shiftsMu.Unlock()

		http.Redirect(w, r, "/shifts", http.StatusSeeOther)
		return
	}
}

func handleDeleteShift(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		idStr := r.FormValue("id")
		id, _ := strconv.ParseInt(idStr, 10, 64)

		shiftsMu.Lock()
		var newShifts []*models.Shift
		for _, s := range shifts {
			if s.ID != id {
				newShifts = append(newShifts, s)
			}
		}
		shifts = newShifts
		shiftsMu.Unlock()

		http.Redirect(w, r, "/shifts", http.StatusSeeOther)
		return
	}
}

func handleAssignRadiologist(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		shiftIDStr := r.FormValue("shift_id")
		radID := r.FormValue("radiologist_id")

		shiftID, _ := strconv.ParseInt(shiftIDStr, 10, 64)

		rosterMu.Lock()
		newEntry := &models.RosterEntry{
			ID: int64(len(roster) + 1),
			ShiftID: shiftID,
			RadiologistID: radID,
			StartDate: time.Now(),
			Status: "active",
		}
		roster = append(roster, newEntry)
		rosterMu.Unlock()

		http.Redirect(w, r, "/shifts", http.StatusSeeOther)
		return
	}
}
