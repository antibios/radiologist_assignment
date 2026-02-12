package assignment

import (
	"radiology-assignment/internal/models"
	"testing"
)

func TestRuleMatches(t *testing.T) {
	e := &Engine{} // ruleMatches only uses helper methods on e, which are stateless or depend on e.calculateStudyAgeMinutes which uses study.IngestTime

	tests := []struct {
		name     string
		rule     *models.AssignmentRule
		study    *models.Study
		expected bool
	}{
		{
			name: "Legacy EQ - Urgency Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{"urgency": "STAT"},
			},
			study:    &models.Study{Urgency: "STAT"},
			expected: true,
		},
		{
			name: "Legacy EQ - Urgency Mismatch",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{"urgency": "STAT"},
			},
			study:    &models.Study{Urgency: "ROUTINE"},
			expected: false,
		},
		{
			name: "New EQ - Urgency Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"urgency": map[string]interface{}{"op": "EQ", "val": "STAT"},
				},
			},
			study:    &models.Study{Urgency: "STAT"},
			expected: true,
		},
		{
			name: "New NEQ - Urgency Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"urgency": map[string]interface{}{"op": "NEQ", "val": "ROUTINE"},
				},
			},
			study:    &models.Study{Urgency: "STAT"},
			expected: true,
		},
		{
			name: "New GT - Age Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"patient_age": map[string]interface{}{"op": "GT", "val": 50},
				},
			},
			study:    &models.Study{PatientAge: 51},
			expected: true,
		},
		{
			name: "New GT - Age Mismatch",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"patient_age": map[string]interface{}{"op": "GT", "val": 50},
				},
			},
			study:    &models.Study{PatientAge: 50},
			expected: false,
		},
		{
			name: "New REGEX - Procedure Code Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"procedure_code": map[string]interface{}{"op": "REGEX", "val": "^CT.*"},
				},
			},
			study:    &models.Study{ProcedureCode: "CTHEAD"},
			expected: true,
		},
		{
			name: "New REGEX - Procedure Code Mismatch",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"procedure_code": map[string]interface{}{"op": "REGEX", "val": "^CT.*"},
				},
			},
			study:    &models.Study{ProcedureCode: "MRHEAD"},
			expected: false,
		},
		{
			name: "List of Conditions - Age Range (GT 10 AND LT 20)",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"patient_age": []interface{}{
						map[string]interface{}{"op": "GT", "val": 10},
						map[string]interface{}{"op": "LT", "val": 20},
					},
				},
			},
			study:    &models.Study{PatientAge: 15},
			expected: true,
		},
		{
			name: "List of Conditions - Age Range Mismatch",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"patient_age": []interface{}{
						map[string]interface{}{"op": "GT", "val": 10},
						map[string]interface{}{"op": "LT", "val": 20},
					},
				},
			},
			study:    &models.Study{PatientAge: 25},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := e.ruleMatches(tt.rule, tt.study); got != tt.expected {
				t.Errorf("ruleMatches() = %v, want %v", got, tt.expected)
			}
		})
	}
}
