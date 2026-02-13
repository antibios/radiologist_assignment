package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"radiology-assignment/internal/assignment"
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

	assignmentsMu sync.RWMutex
	assignments   = []*models.Assignment{
		{ID: 101, StudyID: "ST1001", RadiologistID: "rad1", ShiftID: 1, Strategy: "load_balanced", AssignedAt: time.Now().Add(-5 * time.Minute)},
		{ID: 102, StudyID: "ST1002", RadiologistID: "rad2", ShiftID: 1, Strategy: "primary", AssignedAt: time.Now().Add(-2 * time.Minute)},
		{ID: 103, StudyID: "ST1003", RadiologistID: "rad_vip", ShiftID: 2, Strategy: "special_arrangement", AssignedAt: time.Now().Add(-1 * time.Minute)},
	}

	radiologistWorkload = make(map[string]int64)

	shiftsMu sync.RWMutex
	shifts   = []*models.Shift{
		{ID: 1, Name: "Morning MRI", WorkType: "MRI", Sites: []string{"SiteA"}, PriorityLevel: 1, RequiredCredentials: []string{"MRI"}},
		{ID: 2, Name: "Night CT", WorkType: "CT", Sites: []string{"SiteB"}, PriorityLevel: 2, RequiredCredentials: []string{"CT"}},
	}

	rosterMu sync.RWMutex
	roster   = []*models.RosterEntry{
		{ID: 1, ShiftID: 1, RadiologistID: "rad1", StartDate: time.Now(), Status: "active"},
	}

	proceduresMu sync.RWMutex
	procedures   = []*models.Procedure{
		{ID: 1, Code: "CTHEAD", Description: "CT Head without contrast", Modality: "CT", BodyPart: "Head", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 2, Code: "MRKNEE", Description: "MRI Knee", Modality: "MRI", BodyPart: "Knee", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	configMu sync.RWMutex
	refData  = &models.ReferenceData{
		Sites: []models.Site{
			{Code: "SiteA", Name: "Site A Clinic"},
			{Code: "SiteB", Name: "Site B Hospital"},
		},
		Modalities: []models.Modality{
			{Code: "CT", Name: "Computed Tomography"},
			{Code: "MRI", Name: "Magnetic Resonance Imaging"},
			{Code: "US", Name: "Ultrasound"},
			{Code: "XR", Name: "X-Ray"},
		},
		BodyParts: []models.BodyPart{
			{Name: "Head"},
			{Name: "Chest"},
			{Name: "Abdomen"},
			{Name: "Pelvis"},
			{Name: "Spine"},
			{Name: "Extremity"},
		},
		Credentials: []models.Credential{
			{Code: "CT", Name: "CT Certified"},
			{Code: "MRI", Name: "MRI Certified"},
			{Code: "US", Name: "Ultrasound Certified"},
			{Code: "Neuro", Name: "Neuroradiology"},
			{Code: "MSK", Name: "Musculoskeletal"},
		},
	}

	// Mock Radiologists for assignment
	radiologistsMu sync.RWMutex
	radiologists   = []*models.Radiologist{
		{ID: "rad1", FirstName: "John", LastName: "Doe", MaxConcurrentStudies: 5, Status: "active"},
		{ID: "rad2", FirstName: "Jane", LastName: "Smith", MaxConcurrentStudies: 5, Status: "active"},
		{ID: "rad3", FirstName: "Bob", LastName: "Jones", MaxConcurrentStudies: 5, Status: "active"},
		{ID: "rad_limited", FirstName: "Limited", LastName: "Capacity", MaxConcurrentStudies: 1, Status: "active"},
	}

	radiologistsMap map[string]*models.Radiologist

	// Assignment Engine Instance
	engine *assignment.Engine
)

func init() {
	// Initialize workload map from static assignments
	for _, a := range assignments {
		radiologistWorkload[a.RadiologistID]++
	}
	radiologistsMap = make(map[string]*models.Radiologist)
	for _, rad := range radiologists {
		radiologistsMap[rad.ID] = rad

	}
}

// Implement Interfaces
type InMemoryStore struct{}

func (s *InMemoryStore) GetShiftsByWorkType(ctx context.Context, modality, bodyPart string, site string) ([]*models.Shift, error) {
	shiftsMu.RLock()
	defer shiftsMu.RUnlock()
	var result []*models.Shift
	for _, shift := range shifts {
		// Simple match logic for demo
		if strings.Contains(shift.WorkType, modality) {
			result = append(result, shift)
		}
	}
	return result, nil
}

func (s *InMemoryStore) GetRadiologist(ctx context.Context, id string) (*models.Radiologist, error) {
	if rad, ok := radiologistsMap[id]; ok {
		// Ensure Status is set for logic
		rad.Status = "active"
		return rad, nil
	}
	return nil, errors.New("radiologist not found")
}

func (s *InMemoryStore) GetRadiologists(ctx context.Context, ids []string) ([]*models.Radiologist, error) {
	var result []*models.Radiologist
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	for _, rad := range radiologists {
		if idMap[rad.ID] {
			result = append(result, rad)
		}
	}
	return result, nil
}

func (s *InMemoryStore) GetRadiologistCurrentWorkload(ctx context.Context, radiologistID string) (int64, error) {
	assignmentsMu.RLock()
	defer assignmentsMu.RUnlock()
	return radiologistWorkload[radiologistID], nil
}

func (s *InMemoryStore) GetRadiologistWorkloads(ctx context.Context, radiologistIDs []string) (map[string]int64, error) {
	assignmentsMu.RLock()
	defer assignmentsMu.RUnlock()
	counts := make(map[string]int64)
	targetIDs := make(map[string]bool)
	for _, id := range radiologistIDs {
		counts[id] = 0
		targetIDs[id] = true
	}

	for _, a := range assignments {
		if targetIDs[a.RadiologistID] {
			counts[a.RadiologistID]++
		}
	}
	return counts, nil
}

func (s *InMemoryStore) SaveAssignment(ctx context.Context, a *models.Assignment) error {
	assignmentsMu.Lock()
	defer assignmentsMu.Unlock()
	a.ID = int64(len(assignments) + 1)
	assignments = append(assignments, a)
	radiologistWorkload[a.RadiologistID]++
	return nil
}

type InMemoryRoster struct{}

func (r *InMemoryRoster) GetByShift(shiftID int64) []*models.RosterEntry {
	rosterMu.RLock()
	defer rosterMu.RUnlock()
	var result []*models.RosterEntry
	for _, entry := range roster {
		if entry.ShiftID == shiftID {
			result = append(result, entry)
		}
	}
	return result
}

type InMemoryRules struct{}

func (r *InMemoryRules) GetActive() []*models.AssignmentRule {
	rulesMu.RLock()
	defer rulesMu.RUnlock()
	return rules
}

// Data Structs for UI
type DashboardData struct {
	AssignmentsCount  int
	ActiveRads        int
	PendingStudies    int
	RecentAssignments []*models.Assignment
}

type RulesData struct {
	Rules []*models.AssignmentRule
}

type ShiftsData struct {
	Shifts       []*models.Shift
	Roster       map[int64][]*models.RosterEntry
	Radiologists []*models.Radiologist
	Sites        []models.Site
	Modalities   []models.Modality
	Credentials  []models.Credential
}

type CalendarData struct {
	ViewName       string
	View           string
	Days           []CalendarDay
	UnfilledShifts []CalendarShift
}

type CalendarDay struct {
	Date   time.Time
	Shifts []CalendarShift
}

type CalendarShift struct {
	ShiftName   string
	ShiftType   string
	Date        time.Time
	Filled      bool
	Radiologist string
}

type ProceduresData struct {
	Procedures []*models.Procedure
	Modalities []models.Modality
	BodyParts  []models.BodyPart
}

type ConfigPageData struct {
	*models.ReferenceData
	Radiologists []*models.Radiologist
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize Engine
	engine = assignment.NewEngine(&InMemoryStore{}, &InMemoryRoster{}, &InMemoryRules{})

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

	http.HandleFunc("/procedures", handleProcedures)
	http.HandleFunc("/api/procedures", handleAPIProcedures)
	http.HandleFunc("/api/procedures/edit", handleEditProcedure)
	http.HandleFunc("/api/procedures/delete", handleDeleteProcedure)

	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/api/config/sites", handleAPISites)
	http.HandleFunc("/api/config/sites/edit", handleEditSite)
	http.HandleFunc("/api/config/sites/delete", handleDeleteSite)
	http.HandleFunc("/api/config/modalities", handleAPIModalities)
	http.HandleFunc("/api/config/modalities/edit", handleEditModality)
	http.HandleFunc("/api/config/modalities/delete", handleDeleteModality)
	http.HandleFunc("/api/config/bodyparts", handleAPIBodyParts)
	http.HandleFunc("/api/config/bodyparts/edit", handleEditBodyPart)
	http.HandleFunc("/api/config/bodyparts/delete", handleDeleteBodyPart)
	http.HandleFunc("/api/config/credentials", handleAPICredentials)
	http.HandleFunc("/api/config/credentials/edit", handleEditCredential)
	http.HandleFunc("/api/config/credentials/delete", handleDeleteCredential)

	http.HandleFunc("/api/config/radiologists", handleAPIRadiologists)
	http.HandleFunc("/api/config/radiologists/edit", handleEditRadiologist)
	http.HandleFunc("/api/config/radiologists/delete", handleDeleteRadiologist)

	http.HandleFunc("/calendar", handleCalendar)

	http.HandleFunc("/api/simulate", handleSimulateAssignment)

	log.Printf("API/UI Server started on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func resolveTemplatePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Try going up two levels (for tests running from cmd/api)
		p2 := "../../" + path
		if _, err := os.Stat(p2); err == nil {
			return p2
		}
	}
	return path
}

func toJSON(v interface{}) template.HTML {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return template.HTML(b)
}

func render(w http.ResponseWriter, tmplName string, data interface{}, files ...string) {
	// Include layout in all renders
	var allFiles []string
	allFiles = append(allFiles, resolveTemplatePath("ui/templates/layout.html"))
	for _, f := range files {
		allFiles = append(allFiles, resolveTemplatePath(f))
	}

	tmpl := template.New("layout").Funcs(template.FuncMap{
		"json": toJSON,
	})

	tmpl, err := tmpl.ParseFiles(allFiles...)
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

	assignmentsMu.RLock()
	totalCount := len(assignments)
	// Limit to last 50 assignments for display
	limit := 50
	l := limit
	if l > totalCount {
		l = totalCount
	}
	start := totalCount - l
	recent := make([]*models.Assignment, l)
	copy(recent, assignments[start:])

	// Reverse order for display
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}
	assignmentsMu.RUnlock()

	data := DashboardData{
		AssignmentsCount:  totalCount,
		ActiveRads:        18,
		PendingStudies:    3,
		RecentAssignments: recent,
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

func extractFilters(r *http.Request) map[string]interface{} {
	filters := make(map[string]interface{})

	if val := r.FormValue("filter_urgency"); val != "" {
		filters["urgency"] = val
	}
	if val := r.FormValue("filter_site"); val != "" {
		filters["site"] = val
	}
	if val := r.FormValue("filter_ordering_physician"); val != "" {
		filters["ordering_physician"] = val
	}
	if val := r.FormValue("filter_procedure_code"); val != "" {
		filters["procedure_code"] = val
	}
	if val := r.FormValue("filter_procedure_description"); val != "" {
		filters["procedure_description"] = val
	}
	if val := r.FormValue("filter_prior_location"); val != "" {
		filters["prior_location"] = val
	}
	if val := r.FormValue("filter_technician"); val != "" {
		filters["technician"] = val
	}
	if val := r.FormValue("filter_transcriptionist"); val != "" {
		filters["transcriptionist"] = val
	}
	if val := r.FormValue("filter_patient_age_min"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			filters["patient_age_min"] = v
		}
	}
	if val := r.FormValue("filter_patient_age_max"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			filters["patient_age_max"] = v
		}
	}
	if val := r.FormValue("filter_days_of_week"); val != "" {
		// split by comma
		days := strings.Split(val, ",")
		// trim spaces
		for i := range days {
			days[i] = strings.TrimSpace(days[i])
		}
		filters["days_of_week"] = days
	}

	return filters
}

func handleAPIRules(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		action := r.FormValue("action")
		target := r.FormValue("target")
		filters := extractFilters(r)

		rulesMu.Lock()
		newRule := &models.AssignmentRule{
			ID:               int64(len(rules) + 1),
			Name:             name,
			ActionType:       action,
			ActionTarget:     target,
			ConditionFilters: filters,
			Enabled:          true,
			PriorityOrder:    len(rules) + 1,
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
		target := r.FormValue("target")
		filters := extractFilters(r)

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
				rule.ActionTarget = target
				rule.ConditionFilters = filters
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
	configMu.RLock()

	// Map roster to shift IDs
	rosterMap := make(map[int64][]*models.RosterEntry)
	for _, entry := range roster {
		rosterMap[entry.ShiftID] = append(rosterMap[entry.ShiftID], entry)
	}

	data := ShiftsData{
		Shifts:       shifts,
		Roster:       rosterMap,
		Radiologists: radiologists,
		Sites:        refData.Sites,
		Modalities:   refData.Modalities,
		Credentials:  refData.Credentials,
	}

	configMu.RUnlock()
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
		priorityStr := r.FormValue("priority")
		priority, _ := strconv.Atoi(priorityStr)

		// Capture multi-value fields
		sites := r.Form["sites"]
		creds := r.Form["credentials"]

		shiftsMu.Lock()
		newShift := &models.Shift{
			ID:                  int64(len(shifts) + 1),
			Name:                name,
			WorkType:            workType,
			Sites:               sites,
			PriorityLevel:       priority,
			RequiredCredentials: creds,
			CreatedAt:           time.Now(),
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
			ID:            int64(len(roster) + 1),
			ShiftID:       shiftID,
			RadiologistID: radID,
			StartDate:     time.Now(),
			Status:        "active",
		}
		roster = append(roster, newEntry)
		rosterMu.Unlock()

		http.Redirect(w, r, "/shifts", http.StatusSeeOther)
		return
	}
}

// Procedure Handlers

func handleProcedures(w http.ResponseWriter, r *http.Request) {
	proceduresMu.RLock()
	configMu.RLock()
	data := ProceduresData{
		Procedures: procedures,
		Modalities: refData.Modalities,
		BodyParts:  refData.BodyParts,
	}
	configMu.RUnlock()
	proceduresMu.RUnlock()

	render(w, "procedures", data, "ui/templates/procedures.html")
}

func handleAPIProcedures(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")
		desc := r.FormValue("description")
		modality := r.FormValue("modality")
		bodyPart := r.FormValue("body_part")

		proceduresMu.Lock()
		newProc := &models.Procedure{
			ID:          int64(len(procedures) + 1),
			Code:        code,
			Description: desc,
			Modality:    modality,
			BodyPart:    bodyPart,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		procedures = append(procedures, newProc)
		proceduresMu.Unlock()

		http.Redirect(w, r, "/procedures", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleEditProcedure(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")
		desc := r.FormValue("description")
		modality := r.FormValue("modality")
		bodyPart := r.FormValue("body_part")

		proceduresMu.Lock()
		for _, p := range procedures {
			if p.Code == code {
				p.Description = desc
				p.Modality = modality
				p.BodyPart = bodyPart
				p.UpdatedAt = time.Now()
				break
			}
		}
		proceduresMu.Unlock()

		http.Redirect(w, r, "/procedures", http.StatusSeeOther)
		return
	}
}

func handleDeleteProcedure(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")

		proceduresMu.Lock()
		var newProcs []*models.Procedure
		for _, p := range procedures {
			if p.Code != code {
				newProcs = append(newProcs, p)
			}
		}
		procedures = newProcs
		proceduresMu.Unlock()

		http.Redirect(w, r, "/procedures", http.StatusSeeOther)
		return
	}
}

// Config Handlers

func handleAPIRadiologists(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		id := r.FormValue("id")
		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")
		maxConcurrentStudiesStr := r.FormValue("max_concurrent_studies")
		maxConcurrentStudies, _ := strconv.Atoi(maxConcurrentStudiesStr)
		status := "active" // Default status

		// Capture multi-value fields
		credentials := r.Form["credentials"]
		specialties := r.Form["specialties"]

		radiologistsMu.Lock()
		// Check for duplicate ID
		for _, rad := range radiologists {
			if rad.ID == id {
				radiologistsMu.Unlock()
				http.Error(w, "Radiologist ID already exists", http.StatusBadRequest)
				return
			}
		}

		newRad := &models.Radiologist{
			ID:                   id,
			FirstName:            firstName,
			LastName:             lastName,
			Credentials:          credentials,
			Specialties:          specialties,
			MaxConcurrentStudies: maxConcurrentStudies,
			Status:               status,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		radiologists = append(radiologists, newRad)
		// Update map as well
		radiologistsMap[id] = newRad
		radiologistsMu.Unlock()

		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleEditRadiologist(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		id := r.FormValue("id")
		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")
		maxConcurrentStudiesStr := r.FormValue("max_concurrent_studies")
		maxConcurrentStudies, _ := strconv.Atoi(maxConcurrentStudiesStr)

		// Status might be editable, or default to active
		status := r.FormValue("status")
		if status == "" {
			status = "active"
		}

		credentials := r.Form["credentials"]
		specialties := r.Form["specialties"]

		radiologistsMu.Lock()
		for _, rad := range radiologists {
			if rad.ID == id {
				rad.FirstName = firstName
				rad.LastName = lastName
				rad.MaxConcurrentStudies = maxConcurrentStudies
				rad.Credentials = credentials
				rad.Specialties = specialties
				rad.Status = status
				rad.UpdatedAt = time.Now()
				break
			}
		}
		// Update map not strictly needed if we updated the pointer in slice which is also in map
		radiologistsMu.Unlock()

		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleDeleteRadiologist(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		id := r.FormValue("id")

		radiologistsMu.Lock()
		var newRads []*models.Radiologist
		for _, rad := range radiologists {
			if rad.ID != id {
				newRads = append(newRads, rad)
			}
		}
		radiologists = newRads
		delete(radiologistsMap, id)
		radiologistsMu.Unlock()

		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	configMu.RLock()
	radiologistsMu.RLock()

	// Create a shallow copy of the radiologists slice
	rads := make([]*models.Radiologist, len(radiologists))
	copy(rads, radiologists)

	data := ConfigPageData{
		ReferenceData: refData,
		Radiologists:  rads,
	}
	radiologistsMu.RUnlock()
	configMu.RUnlock()

	render(w, "config", data, "ui/templates/config.html")
}

func handleAPISites(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		name := r.FormValue("name")
		configMu.Lock()
		refData.Sites = append(refData.Sites, models.Site{Code: code, Name: name})
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleDeleteSite(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		code := r.FormValue("code")
		configMu.Lock()
		newSites := []models.Site{}
		for _, s := range refData.Sites {
			if s.Code != code {
				newSites = append(newSites, s)
			}
		}
		refData.Sites = newSites
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
}

func handleEditSite(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		originalCode := r.FormValue("original_code")
		code := r.FormValue("code")
		name := r.FormValue("name")

		configMu.Lock()
		for i, s := range refData.Sites {
			if s.Code == originalCode {
				refData.Sites[i].Code = code
				refData.Sites[i].Name = name
				break
			}
		}
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleAPIModalities(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		code := r.FormValue("code")
		name := r.FormValue("name")
		configMu.Lock()
		refData.Modalities = append(refData.Modalities, models.Modality{Code: code, Name: name})
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
}

func handleDeleteModality(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		code := r.FormValue("code")
		configMu.Lock()
		newMods := []models.Modality{}
		for _, m := range refData.Modalities {
			if m.Code != code {
				newMods = append(newMods, m)
			}
		}
		refData.Modalities = newMods
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
}

func handleEditModality(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		originalCode := r.FormValue("original_code")
		code := r.FormValue("code")
		name := r.FormValue("name")

		configMu.Lock()
		for i, m := range refData.Modalities {
			if m.Code == originalCode {
				refData.Modalities[i].Code = code
				refData.Modalities[i].Name = name
				break
			}
		}
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleAPIBodyParts(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		name := r.FormValue("name")
		configMu.Lock()
		refData.BodyParts = append(refData.BodyParts, models.BodyPart{Name: name})
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
}

func handleDeleteBodyPart(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		name := r.FormValue("name")
		configMu.Lock()
		newBPs := []models.BodyPart{}
		for _, bp := range refData.BodyParts {
			if bp.Name != name {
				newBPs = append(newBPs, bp)
			}
		}
		refData.BodyParts = newBPs
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
}

func handleEditBodyPart(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		originalName := r.FormValue("original_name")
		name := r.FormValue("name")

		configMu.Lock()
		for i, bp := range refData.BodyParts {
			if bp.Name == originalName {
				refData.BodyParts[i].Name = name
				break
			}
		}
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleAPICredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		code := r.FormValue("code")
		name := r.FormValue("name")
		configMu.Lock()
		refData.Credentials = append(refData.Credentials, models.Credential{Code: code, Name: name})
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
}

func handleEditCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		originalCode := r.FormValue("original_code")
		code := r.FormValue("code")
		name := r.FormValue("name")

		configMu.Lock()
		for i, c := range refData.Credentials {
			if c.Code == originalCode {
				refData.Credentials[i].Code = code
				refData.Credentials[i].Name = name
				break
			}
		}
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		code := r.FormValue("code")
		configMu.Lock()
		newCreds := []models.Credential{}
		for _, c := range refData.Credentials {
			if c.Code != code {
				newCreds = append(newCreds, c)
			}
		}
		refData.Credentials = newCreds
		configMu.Unlock()
		http.Redirect(w, r, "/config", http.StatusSeeOther)
		return
	}
}

func handleCalendar(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "month"
	}

	now := time.Now()
	var days []CalendarDay
	var unfilled []CalendarShift

	shiftsMu.RLock()
	rosterMu.RLock()

	// Determine date range based on view
	// Simply taking current day for 'day', current week for 'week', current month for 'month'
	var start, end time.Time

	switch view {
	case "day":
		start = now
		end = now
	case "week":
		// Find start of week (Sunday)
		weekday := int(now.Weekday())
		start = now.AddDate(0, 0, -weekday)
		end = start.AddDate(0, 0, 6)
	case "month":
		// Find start of month
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, -1)
	}

	// Generate grid
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		day := CalendarDay{Date: d}

		for _, s := range shifts {
			// Check if filled
			filled := false
			radName := ""
			for _, entry := range roster {
				if entry.ShiftID == s.ID {
					// Check if date falls in roster entry range
					// Simple check: StartDate only for now as defined in roster logic
					// Assuming daily roster entries or handling logic
					// Let's match Year/Month/Day
					if entry.StartDate.Year() == d.Year() && entry.StartDate.Month() == d.Month() && entry.StartDate.Day() == d.Day() {
						filled = true
						radName = entry.RadiologistID // Should map to name via lookups
						break
					}
				}
			}

			cs := CalendarShift{
				ShiftName:   s.Name,
				ShiftType:   s.WorkType,
				Date:        d,
				Filled:      filled,
				Radiologist: radName,
			}
			day.Shifts = append(day.Shifts, cs)

			if !filled {
				unfilled = append(unfilled, cs)
			}
		}
		days = append(days, day)
	}

	rosterMu.RUnlock()
	shiftsMu.RUnlock()

	data := CalendarData{
		ViewName:       strings.Title(view),
		View:           view,
		Days:           days,
		UnfilledShifts: unfilled,
	}

	render(w, "calendar", data, "ui/templates/calendar.html")
}

func handleSimulateAssignment(w http.ResponseWriter, r *http.Request) {
	log.Println("handleSimulateAssignment: start")
	defer log.Println("handleSimulateAssignment: end")
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		studyID := r.FormValue("study_id")
		modality := r.FormValue("modality")
		procedureCode := r.FormValue("procedure_code")
		orderingPhysician := r.FormValue("ordering_physician")
		patientAgeStr := r.FormValue("patient_age")
		patientAge, _ := strconv.Atoi(patientAgeStr)

		ingestTimeStr := r.FormValue("ingest_time")
		ingestTime := time.Now()
		if ingestTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, ingestTimeStr); err == nil {
				ingestTime = t
			}
		}

		// New fields
		urgency := r.FormValue("urgency")
		site := r.FormValue("site")
		procedureDesc := r.FormValue("procedure_description")
		priorLocation := r.FormValue("prior_location")
		technician := r.FormValue("technician")
		transcriptionist := r.FormValue("transcriptionist")

		// Default site if not provided
		if site == "" {
			site = "SiteA"
		}

		study := &models.Study{
			ID:                   studyID,
			Modality:             modality,
			BodyPart:             "General",
			Site:                 site,
			ProcedureCode:        procedureCode,
			OrderingPhysician:    orderingPhysician,
			PatientAge:           patientAge,
			Timestamp:            ingestTime.Format("20060102150405"),
			IngestTime:           ingestTime,
			Urgency:              urgency,
			ProcedureDescription: procedureDesc,
			PriorLocation:        priorLocation,
			Technician:           technician,
			Transcriptionist:     transcriptionist,
		}

		assignment, err := engine.Assign(context.Background(), study)
		if err != nil {
			http.Error(w, fmt.Sprintf("Assignment Failed: %v", err), http.StatusServiceUnavailable)
			return
		}

		if assignment.RadiologistID == "WORKLIST" {
			fmt.Fprintf(w, "Assigned to Worklist: %s", assignment.Strategy)
		} else {
			fmt.Fprintf(w, "Assigned to %s", assignment.RadiologistID)
		}
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}
