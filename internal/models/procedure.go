package models

import "time"

type Procedure struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Modality    string    `json:"modality"`
	BodyPart    string    `json:"body_part"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
