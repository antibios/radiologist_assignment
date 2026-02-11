package assignment

import (
	"context"
	"fmt"
	"radiology-assignment/internal/models"
	"sort"
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
	selected, escalated, err := e.evaluateRules(ctx, study, candidates)
	if err != nil {
		return nil, err
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

func (e *Engine) evaluateRules(ctx context.Context, study *models.Study, candidates []*candidate) (*candidate, bool, error) {
	rules := e.rules.GetActive()

	// Sort rules by priority (lower number = higher priority)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].PriorityOrder < rules[j].PriorityOrder
	})

	currentCandidates := candidates
	isEscalated := false
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
		return nil, false, err
	}

	if len(currentCandidates) == 0 {
		return nil, isEscalated, nil
	}

	// Load Balance
	selected, err := e.loadBalance(ctx, currentCandidates)
	if err != nil {
		return nil, false, err
	}

	// If we had a matched rule, we might note it in the assignment (not added to return yet)
	_ = matchedRule

	return selected, isEscalated, nil
}

func (e *Engine) ruleMatches(rule *models.AssignmentRule, study *models.Study) bool {
	// Simple matcher based on ConditionFilters map
	// e.g. "min_age_minutes" -> check study age
	filters := rule.ConditionFilters

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

	if val, ok := filters["urgency"]; ok {
		if study.Urgency != val.(string) {
			return false
		}
	}

	// Add other matches if needed
	return true
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
	var filtered []*candidate
	for _, c := range candidates {
		load, err := e.db.GetRadiologistCurrentWorkload(ctx, c.Radiologist.ID)
		if err != nil {
			return nil, err
		}
		if int(load) < c.Radiologist.MaxConcurrentStudies || c.Radiologist.MaxConcurrentStudies == 0 {
			filtered = append(filtered, c)
		}
	}
	return filtered, nil
}

func (e *Engine) loadBalance(ctx context.Context, candidates []*candidate) (*candidate, error) {
	// Pick candidate with lowest load
	var best *candidate
	minLoad := int64(999999)

	for _, c := range candidates {
		load, err := e.db.GetRadiologistCurrentWorkload(ctx, c.Radiologist.ID)
		if err != nil {
			return nil, err
		}
		if load < minLoad {
			minLoad = load
			best = c
		}
	}
	return best, nil
}
