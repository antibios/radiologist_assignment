package assignment

import (
	"context"
	"fmt"
	"radiology-assignment/internal/models"
	"testing"
)

// FR-4.3.2: The system shall support at least 1,000 active assignment rules.
// NFR-5.1.3: Rule evaluation shall complete within acceptable latency across 1,000+ rules.
func BenchmarkRuleEvaluation_1000Rules(b *testing.B) {
	// Setup 1000 rules
	rules := make([]*models.AssignmentRule, 1000)
	for i := 0; i < 1000; i++ {
		rules[i] = &models.AssignmentRule{
			ID: int64(i),
			PriorityOrder: i,
			// Add a condition that fails so we iterate through them
			ConditionFilters: map[string]interface{}{"modality": "CT"},
			ActionType: "FILTER_COMPETENCY",
		}
	}

	// Setup simple study and roster
	study := &models.Study{ID: "bench_study", Modality: "MRI"}
	shift := &models.Shift{ID: 1}
	rad := &models.Radiologist{ID: "rad1", Status: "active", Credentials: []string{"MRI"}}

	engine := setupEngine(b, []*models.Shift{shift}, []*models.Radiologist{rad}, map[int64][]string{1: {"rad1"}}, rules)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Assign(context.Background(), study)
	}
}

// NFR-5.2.3: The system shall evaluate up to 1,500 user/shift combinations per assignment cycle.
func BenchmarkAssignment_LargeRoster(b *testing.B) {
	numRads := 1500
	rads := make([]*models.Radiologist, numRads)
	rosterMap := make(map[int64][]string)
	rosterList := make([]string, numRads)

	for i := 0; i < numRads; i++ {
		id := fmt.Sprintf("rad%d", i)
		rads[i] = &models.Radiologist{ID: id, Status: "active", Credentials: []string{"MRI"}}
		rosterList[i] = id
	}
	rosterMap[1] = rosterList

	shift := &models.Shift{ID: 1, WorkType: "MRI"}
	study := &models.Study{ID: "bench_study", Modality: "MRI"}

	engine := setupEngine(b, []*models.Shift{shift}, rads, rosterMap, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Assign(context.Background(), study)
	}
}

// NFR-5.1.1: Assignment decision latency shall not exceed 500ms (p95) from study submission to assignment.
func BenchmarkAssign_EndToEnd(b *testing.B) {
	// Mixed scenario: 50 rules, 50 radiologists
	numRules := 50
	rules := make([]*models.AssignmentRule, numRules)
	for i := 0; i < numRules; i++ {
		rules[i] = &models.AssignmentRule{
			ID: int64(i),
			PriorityOrder: i,
			ConditionFilters: map[string]interface{}{"min_age_minutes": 1000}, // Won't match
			ActionType: "ESCALATE",
		}
	}

	numRads := 50
	rads := make([]*models.Radiologist, numRads)
	rosterList := make([]string, numRads)
	for i := 0; i < numRads; i++ {
		id := fmt.Sprintf("rad%d", i)
		rads[i] = &models.Radiologist{ID: id, Status: "active", Credentials: []string{"MRI"}}
		rosterList[i] = id
	}
	rosterMap := map[int64][]string{1: rosterList}

	shift := &models.Shift{ID: 1, WorkType: "MRI"}
	study := &models.Study{ID: "bench_study", Modality: "MRI"}

	engine := setupEngine(b, []*models.Shift{shift}, rads, rosterMap, rules)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Assign(context.Background(), study)
	}
}
