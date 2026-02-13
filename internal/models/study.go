package models

import "time"

type Study struct {
	ID                   string    `json:"id"`
	MessageID            string    `json:"message_id"`
	Site                 string    `json:"site"`
	Timestamp            string    `json:"timestamp"` // HL7 timestamp format
	Modality             string    `json:"modality"`
	BodyPart             string    `json:"body_part"`
	Urgency              string    `json:"urgency"`
	Indication           string    `json:"indication"`
	ProcedureCode        string    `json:"procedure_code"`
	ProcedureDescription string    `json:"procedure_description"`
	OrderingPhysician    string    `json:"ordering_physician"`
	PatientAge           int       `json:"patient_age"`
	IngestTime           time.Time `json:"ingest_time"` // Added for internal tracking
	PriorLocation        string    `json:"prior_location"`
	Technician           string    `json:"technician"`
	Transcriptionist     string    `json:"transcriptionist"`
	RVU                  float64   `json:"rvu"`
}

func (s *Study) GetExamTime() time.Time {
	if s.Timestamp == "" {
		return s.IngestTime
	}
	// Try parsing HL7 format YYYYMMDDHHMMSS
	t, err := time.Parse("20060102150405", s.Timestamp)
	if err == nil {
		return t
	}
	return s.IngestTime
}
