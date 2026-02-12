package assignment

import (
	"radiology-assignment/internal/models"
	"testing"
	"time"
)

func TestRuleMatches_ExtendedFields(t *testing.T) {
	e := &Engine{}

	tests := []struct {
		name       string
		rule       *models.AssignmentRule
		study      *models.Study
		wantMatch  bool
	}{
		{
			name: "Procedure Code Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{"procedure_code": "CTHEAD"},
			},
			study:     &models.Study{ProcedureCode: "CTHEAD"},
			wantMatch: true,
		},
		{
			name: "Procedure Code Mismatch",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{"procedure_code": "CTHEAD"},
			},
			study:     &models.Study{ProcedureCode: "CTABD"},
			wantMatch: false,
		},
		{
			name: "Ordering Physician Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{"ordering_physician": "Dr. Smith"},
			},
			study:     &models.Study{OrderingPhysician: "Dr. Smith"},
			wantMatch: true,
		},
		{
			name: "Patient Age Range Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"patient_age_min": 10,
					"patient_age_max": 20,
				},
			},
			study:     &models.Study{PatientAge: 15},
			wantMatch: true,
		},
		{
			name: "Patient Age Too Young",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"patient_age_min": 10,
				},
			},
			study:     &models.Study{PatientAge: 5},
			wantMatch: false,
		},
		{
			name: "Patient Age Too Old",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"patient_age_max": 65,
				},
			},
			study:     &models.Study{PatientAge: 70},
			wantMatch: false,
		},
		{
			name: "Time Range Match (Day)",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"exam_time_range": "08:00-12:00",
				},
			},
			study:     &models.Study{Timestamp: "20231010090000"}, // 09:00
			wantMatch: true,
		},
		{
			name: "Time Range Mismatch (Day)",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"exam_time_range": "08:00-12:00",
				},
			},
			study:     &models.Study{Timestamp: "20231010130000"}, // 13:00
			wantMatch: false,
		},
		{
			name: "Time Range Match (Overnight)",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"exam_time_range": "22:00-06:00",
				},
			},
			study:     &models.Study{Timestamp: "20231010230000"}, // 23:00
			wantMatch: true,
		},
		{
			name: "Time Range Match (Overnight Early Morning)",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"exam_time_range": "22:00-06:00",
				},
			},
			study:     &models.Study{Timestamp: "20231011050000"}, // 05:00
			wantMatch: true,
		},
		{
			name: "Day of Week Match",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"days_of_week": []string{"Monday", "Wednesday"},
				},
			},
			study: &models.Study{
				// 2023-10-09 is a Monday
				Timestamp: "20231009100000",
			},
			wantMatch: true,
		},
		{
			name: "Day of Week Mismatch",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"days_of_week": []string{"Tuesday", "Thursday"},
				},
			},
			study: &models.Study{
				// 2023-10-09 is a Monday
				Timestamp: "20231009100000",
			},
			wantMatch: false,
		},
		{
			name: "Day of Week Interface Slice",
			rule: &models.AssignmentRule{
				ConditionFilters: map[string]interface{}{
					"days_of_week": []interface{}{"Monday", "Wednesday"},
				},
			},
			study: &models.Study{
				Timestamp: "20231009100000",
			},
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.ruleMatches(tt.rule, tt.study)
			if got != tt.wantMatch {
				t.Errorf("ruleMatches() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestMatchesTimeRange(t *testing.T) {
	e := &Engine{}

	layout := "15:04"
	parseTime := func(s string) time.Time {
		t, _ := time.Parse(layout, s)
		return t
	}

	tests := []struct {
		current string
		rangeStr string
		want    bool
	}{
		{"09:00", "08:00-12:00", true},
		{"08:00", "08:00-12:00", true},
		{"12:00", "08:00-12:00", true},
		{"07:59", "08:00-12:00", false},
		{"12:01", "08:00-12:00", false},
		{"23:00", "22:00-06:00", true},
		{"05:00", "22:00-06:00", true},
		{"12:00", "22:00-06:00", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+" in "+tt.rangeStr, func(t *testing.T) {
			if got := e.matchesTimeRange(parseTime(tt.current), tt.rangeStr); got != tt.want {
				t.Errorf("matchesTimeRange() = %v, want %v", got, tt.want)
			}
		})
	}
}
