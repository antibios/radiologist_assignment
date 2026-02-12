package assignment

import (
	"context"
	"fmt"
	"radiology-assignment/internal/models"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Engine struct {
	db     DataStore
	roster RosterService
	rules  RulesService
}

func NewEngine(db DataStore, roster RosterService, rules RulesService) *Engine {
	return &Engine{
		db:     db,
		roster: roster,
		rules:  rules,
	}
}

type candidate struct {
	Radiologist *models.Radiologist
	ShiftID     int64
	CurrentLoad int64
}

func (e *Engine) Assign(ctx context.Context, study *models.Study) (*models.Assignment, error) {
	if study == nil {
		return nil, fmt.Errorf("study cannot be nil")
	}

	// Step 1: Match shifts based on study characteristics
	shifts, err := e.matchShifts(ctx, study)
	if err != nil {
		return nil, err
	}
	if len(shifts) == 0 {
		return nil, fmt.Errorf("no matching shifts for study %s", study.ID)
	}

	// Step 2: Resolve radiologists from roster for matched shifts
	candidates, err := e.resolveRadiologists(ctx, shifts)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available radiologists for shifts")
	}

	// Step 3: Apply rule-based assignment pipeline
	selected, worklistTarget, escalated, err := e.evaluateRules(ctx, study, candidates)
	if err != nil {
		return nil, err
	}

	if worklistTarget != "" {
		return &models.Assignment{
			StudyID:       study.ID,
			RadiologistID: "WORKLIST",
			ShiftID:       0,
			AssignedAt:    study.IngestTime,
			Escalated:     escalated,
			Strategy:      worklistTarget,
		}, nil
	}

	if selected == nil {
		return nil, fmt.Errorf("no candidate selected after rule evaluation")
	}

	assignment := &models.Assignment{
		StudyID:       study.ID,
		RadiologistID: selected.Radiologist.ID,
		ShiftID:       selected.ShiftID,
		AssignedAt:    study.IngestTime, // Should be Now(), but using IngestTime for simplicity or mock it
		Escalated:     escalated,
		Strategy:      "load_balanced", // Default
	}

	// Save assignment (optional step in logic flow, but good for completeness)
	if err := e.db.SaveAssignment(ctx, assignment); err != nil {
		return nil, err
	}

	return assignment, nil
}

func (e *Engine) matchShifts(ctx context.Context, study *models.Study) ([]*models.Shift, error) {
	return e.db.GetShiftsByWorkType(ctx, study.Modality, study.BodyPart, study.Site)
}

func (e *Engine) resolveRadiologists(ctx context.Context, shifts []*models.Shift) ([]*candidate, error) {
	radShiftMap := make(map[string]int64)
	var uniqueIDs []string

	for _, shift := range shifts {
		entries := e.roster.GetByShift(shift.ID)
		for _, entry := range entries {
			if _, exists := radShiftMap[entry.RadiologistID]; !exists {
				radShiftMap[entry.RadiologistID] = shift.ID
				uniqueIDs = append(uniqueIDs, entry.RadiologistID)
			}
		}
	}

	if len(uniqueIDs) == 0 {
		return []*candidate{}, nil
	}

	radiologists, err := e.db.GetRadiologists(ctx, uniqueIDs)
	if err != nil {
		return nil, err
	}

	var result []*candidate
	for _, rad := range radiologists {
		if rad.Status != "active" {
			continue
		}

		shiftID, ok := radShiftMap[rad.ID]
		if !ok {
			continue
		}

		result = append(result, &candidate{
			Radiologist: rad,
			ShiftID:     shiftID,
		})
	}

	return result, nil
}

func (e *Engine) evaluateRules(ctx context.Context, study *models.Study, candidates []*candidate) (*candidate, string, bool, error) {
	rules := e.rules.GetActive()

	// Sort rules by priority (lower number = higher priority)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].PriorityOrder < rules[j].PriorityOrder
	})

	currentCandidates := candidates
	isEscalated := false
	var worklistTarget string
	var matchedRule *models.AssignmentRule

	for _, rule := range rules {
		if !e.ruleMatches(rule, study) {
			continue
		}

		matchedRule = rule // Keep track of last matched rule

		switch rule.ActionType {
		case "FILTER_COMPETENCY":
			// Simple implementation: Filter candidates who have credentials matching study.Modality
			currentCandidates = e.filterByCompetency(currentCandidates, study.Modality)

		case "ASSIGN_TO_RADIOLOGIST":
			// Filter specifically for this radiologist
			if target := rule.ActionTarget; target != "" {
				currentCandidates = e.filterByRadiologistID(currentCandidates, target)
			}

		case "ASSIGN_TO_SHIFT":
			if target := rule.ActionTarget; target != "" {
				shiftID, err := strconv.ParseInt(target, 10, 64)
				if err == nil {
					currentCandidates = e.filterByShiftID(currentCandidates, shiftID)
				}
			}

		case "ASSIGN_TO_WORKLIST":
			worklistTarget = rule.ActionTarget
			// If assigned to worklist, we stop processing candidates and return immediately
			return nil, worklistTarget, isEscalated, nil

		case "ESCALATE":
			isEscalated = true
		}
	}

	// Filter by Capacity
	// If escalated, maybe we ignore capacity? Requirements say "Reassign to available radiologists with higher priority".
	// But let's assume capacity limits still apply unless forced.
	// FR-4.6.3: "respect maximum concurrent study limits per radiologist if configured"
	// Let's filter by capacity.
	currentCandidates, err := e.filterByCapacity(ctx, currentCandidates)
	if err != nil {
		return nil, "", false, err
	}

	if len(currentCandidates) == 0 {
		return nil, "", isEscalated, nil
	}

	// Load Balance
	selected, err := e.loadBalance(ctx, currentCandidates)
	if err != nil {
		return nil, "", false, err
	}

	// If we had a matched rule, we might note it in the assignment (not added to return yet)
	_ = matchedRule

	return selected, "", isEscalated, nil
}

func (e *Engine) ruleMatches(rule *models.AssignmentRule, study *models.Study) bool {
	// Matcher based on ConditionFilters map
	filters := rule.ConditionFilters

	// 1. Min Age (Wait time)
	if val, ok := filters["min_age_minutes"]; ok {
		var minAge float64
		switch v := val.(type) {
		case int:
			minAge = float64(v)
		case float64:
			minAge = v
		case int64:
			minAge = float64(v)
		}

		age := e.calculateStudyAgeMinutes(study)
		if age <= minAge {
			return false
		}
	}

	// 2. Urgency
	if val, ok := filters["urgency"]; ok {
		if study.Urgency != val.(string) {
			return false
		}
	}

	// 3. Procedure Code
	if val, ok := filters["procedure_code"]; ok {
		if study.ProcedureCode != val.(string) {
			return false
		}
	}

	// 4. Body Part
	if val, ok := filters["body_part"]; ok {
		if study.BodyPart != val.(string) {
			return false
		}
	}

	// 5. Ordering Physician
	if val, ok := filters["ordering_physician"]; ok {
		if study.OrderingPhysician != val.(string) {
			return false
		}
	}

	// 6. Site
	if val, ok := filters["site"]; ok {
		if study.Site != val.(string) {
			return false
		}
	}

	// 7. Patient Age Range
	if val, ok := filters["patient_age_min"]; ok {
		min := toFloat(val)
		if float64(study.PatientAge) < min {
			return false
		}
	}
	if val, ok := filters["patient_age_max"]; ok {
		max := toFloat(val)
		if float64(study.PatientAge) > max {
			return false
		}
	}

	// 8. Exam Time Range (HH:MM-HH:MM)
	if val, ok := filters["exam_time_range"]; ok {
		if !e.matchesTimeRange(study.GetExamTime(), val.(string)) {
			return false
		}
	}

	// 9. Day of Week
	if val, ok := filters["days_of_week"]; ok {
		if !e.matchesDayOfWeek(study.GetExamTime(), val) {
			return false
		}
	}

	// 10. Procedure Description
	if val, ok := filters["procedure_description"]; ok {
		if study.ProcedureDescription != val.(string) {
			return false
		}
	}

	// 11. Prior Location
	if val, ok := filters["prior_location"]; ok {
		if study.PriorLocation != val.(string) {
			return false
		}
	}

	// 12. Technician
	if val, ok := filters["technician"]; ok {
		if study.Technician != val.(string) {
			return false
		}
	}

	// 13. Transcriptionist
	if val, ok := filters["transcriptionist"]; ok {
		if study.Transcriptionist != val.(string) {
			return false
		}
	}

	return true
}

func toFloat(val interface{}) float64 {
	switch v := val.(type) {
	case int:
		return float64(v)
	case float64:
		return v
	case int64:
		return float64(v)
	}
	return 0
}

func (e *Engine) matchesTimeRange(t time.Time, rangeStr string) bool {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return false
	}
	startStr, endStr := parts[0], parts[1]

	layout := "15:04"
	start, err := time.Parse(layout, startStr)
	if err != nil {
		return false
	}
	end, err := time.Parse(layout, endStr)
	if err != nil {
		return false
	}

	// Normalize t to just time part for comparison
	currentStr := t.Format(layout)
	current, _ := time.Parse(layout, currentStr)

	// Handle overnight ranges e.g. 22:00-06:00
	if end.Before(start) {
		return !current.Before(start) || !current.After(end)
	}

	return (current.Equal(start) || current.After(start)) && (current.Equal(end) || current.Before(end))
}

func (e *Engine) matchesDayOfWeek(t time.Time, daysVal interface{}) bool {
	day := t.Weekday().String() // "Monday", "Tuesday", etc.
	// Allow 3-letter abbreviations too? Let's check against what's provided.

	var allowedDays []string
	switch v := daysVal.(type) {
	case []string:
		allowedDays = v
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				allowedDays = append(allowedDays, s)
			}
		}
	}

	for _, d := range allowedDays {
		if strings.EqualFold(d, day) || strings.EqualFold(d, day[:3]) {
			return true
		}
	}
	return false
}

func (e *Engine) calculateStudyAgeMinutes(study *models.Study) float64 {
	// Use IngestTime if available
	if !study.IngestTime.IsZero() {
		return time.Since(study.IngestTime).Minutes()
	}
	return 0
}

func (e *Engine) filterByCompetency(candidates []*candidate, requiredCredential string) []*candidate {
	var filtered []*candidate
	for _, c := range candidates {
		hasCred := false
		for _, cred := range c.Radiologist.Credentials {
			if cred == requiredCredential {
				hasCred = true
				break
			}
		}
		if hasCred {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (e *Engine) filterByShiftID(candidates []*candidate, shiftID int64) []*candidate {
	var filtered []*candidate
	for _, c := range candidates {
		if c.ShiftID == shiftID {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (e *Engine) filterByRadiologistID(candidates []*candidate, targetID string) []*candidate {
	var filtered []*candidate
	for _, c := range candidates {
		if c.Radiologist.ID == targetID {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (e *Engine) filterByCapacity(ctx context.Context, candidates []*candidate) ([]*candidate, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	ids := make([]string, len(candidates))
	for i, c := range candidates {
		ids[i] = c.Radiologist.ID
	}

	workloads, err := e.db.GetRadiologistWorkloads(ctx, ids)
	if err != nil {
		return nil, err
	}

	var filtered []*candidate
	for _, c := range candidates {
		load := workloads[c.Radiologist.ID]
		c.CurrentLoad = load
		if int(load) < c.Radiologist.MaxConcurrentStudies || c.Radiologist.MaxConcurrentStudies == 0 {
			filtered = append(filtered, c)
		}
	}
	return filtered, nil
}

func (e *Engine) loadBalance(ctx context.Context, candidates []*candidate) (*candidate, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	// Workloads are already cached in candidates from filterByCapacity

	// Pick candidate with lowest load
	var best *candidate
	minLoad := int64(999999)

	for _, c := range candidates {
		load := c.CurrentLoad
		if load < minLoad {
			minLoad = load
			best = c
		}
	}
	return best, nil
}
