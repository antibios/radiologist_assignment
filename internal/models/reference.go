package models

type ReferenceData struct {
	Sites       []Site       `json:"sites"`
	Modalities  []Modality   `json:"modalities"`
	BodyParts   []BodyPart   `json:"body_parts"`
	Credentials []Credential `json:"credentials"`
}

type Site struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Modality struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type BodyPart struct {
	Name string `json:"name"`
}

type Credential struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
