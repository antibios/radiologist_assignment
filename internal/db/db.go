package db

import (
	"context"
	"database/sql"
	"time"
    "github.com/lib/pq"
)

type Radiologist struct {
	ID                   string
	FirstName            string
	LastName             string
	MaxConcurrentStudies int32
	Credentials          []string
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Shift struct {
	ID                  int64
	Name                string
	WorkType            string
	Sites               []string
	PriorityLevel       int32
	RequiredCredentials []string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type RosterEntry struct {
	ID            int64
	ShiftID       int64
	RadiologistID string
	StartDate     time.Time
	Status        string
	CreatedAt     time.Time
}

type AssignmentRule struct {
	ID               int64
	Name             string
	PriorityOrder    int32
	ActionType       string
	ConditionFilters []byte // JSONB
	Enabled          bool
	CreatedAt        time.Time
}

type Assignment struct {
	ID            int64
	StudyID       string
	RadiologistID string
	ShiftID       int64
	Strategy      string
	AssignedAt    time.Time
}

// Queries interface mimicking sqlc generated code
type Queries struct {
	db *sql.DB
}

func New(db *sql.DB) *Queries {
	return &Queries{db: db}
}

func (q *Queries) ListRadiologists(ctx context.Context) ([]Radiologist, error) {
	rows, err := q.db.QueryContext(ctx, "SELECT id, first_name, last_name, max_concurrent_studies, credentials, status, created_at, updated_at FROM radiologists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Radiologist
	for rows.Next() {
		var i Radiologist
		if err := rows.Scan(&i.ID, &i.FirstName, &i.LastName, &i.MaxConcurrentStudies, pq.Array(&i.Credentials), &i.Status, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, nil
}

func (q *Queries) CreateRadiologist(ctx context.Context, arg Radiologist) error {
    _, err := q.db.ExecContext(ctx,
        "INSERT INTO radiologists (id, first_name, last_name, max_concurrent_studies, credentials, status) VALUES ($1, $2, $3, $4, $5, $6)",
        arg.ID, arg.FirstName, arg.LastName, arg.MaxConcurrentStudies, pq.Array(arg.Credentials), arg.Status,
    )
    return err
}

// ... Add other CRUD methods similarly as needed by the Store implementation
// For brevity in this task, implementing the core methods required for the E2E flow
