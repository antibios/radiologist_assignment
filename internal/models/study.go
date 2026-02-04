package models

import "time"

type Study struct {
	ID         string    `json:"id"`
	MessageID  string    `json:"message_id"`
	Site       string    `json:"site"`
	Timestamp  string    `json:"timestamp"` // HL7 timestamp format
	Modality   string    `json:"modality"`
	BodyPart   string    `json:"body_part"`
	Urgency    string    `json:"urgency"`
	Indication string    `json:"indication"`
	IngestTime time.Time `json:"ingest_time"` // Added for internal tracking
}
