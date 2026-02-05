package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"radiology-assignment/internal/db"
	"radiology-assignment/internal/models"
    "github.com/lib/pq"
)

type PostgresStore struct {
	q  *db.Queries
	db *sql.DB // Keep reference to raw DB for direct queries in simulation
}

func NewPostgresStore(conn *sql.DB) *PostgresStore {
	return &PostgresStore{q: db.New(conn), db: conn}
}

// Ensure interface implementation
// Note: We need to adapt the db package structs to internal/models structs if they differ,
// or update internal/models to match db package.
// For simplicity, let's assume we map them here.

func (s *PostgresStore) GetShiftsByWorkType(ctx context.Context, modality, bodyPart string, site string) ([]*models.Shift, error) {
	// Implement query logic using s.q.db directly if specific query not in `db` package wrapper
    // In a real sqlc world, we'd have a generated method `GetShiftsBy...`

    // Manual implementation for now since we are simulating sqlc
    rows, err := s.q.db.QueryContext(ctx, "SELECT id, name, work_type, sites, priority_level, required_credentials FROM shifts WHERE work_type LIKE $1", "%"+modality+"%")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var shifts []*models.Shift
    for rows.Next() {
        var s models.Shift
        var sites []string
        var creds []string
        if err := rows.Scan(&s.ID, &s.Name, &s.WorkType, pq.Array(&sites), &s.PriorityLevel, pq.Array(&creds)); err != nil {
            return nil, err
        }
        s.Sites = sites
        s.RequiredCredentials = creds
        shifts = append(shifts, &s)
    }
    return shifts, nil
}

func (s *PostgresStore) GetRadiologist(ctx context.Context, id string) (*models.Radiologist, error) {
    row := s.q.db.QueryRowContext(ctx, "SELECT id, first_name, last_name, max_concurrent_studies, credentials, status FROM radiologists WHERE id = $1", id)
    var r models.Radiologist
    var creds []string
    if err := row.Scan(&r.ID, &r.FirstName, &r.LastName, &r.MaxConcurrentStudies, pq.Array(&creds), &r.Status); err != nil {
        return nil, err
    }
    r.Credentials = creds
    return &r, nil
}

func (s *PostgresStore) GetRadiologistCurrentWorkload(ctx context.Context, radiologistID string) (int64, error) {
    var count int64
    err := s.q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM assignments WHERE radiologist_id = $1", radiologistID).Scan(&count)
    return count, err
}

func (s *PostgresStore) SaveAssignment(ctx context.Context, a *models.Assignment) error {
    _, err := s.q.db.ExecContext(ctx, "INSERT INTO assignments (study_id, radiologist_id, shift_id, strategy, assigned_at) VALUES ($1, $2, $3, $4, $5)",
        a.StudyID, a.RadiologistID, a.ShiftID, a.Strategy, a.AssignedAt)
    return err
}

// Roster Service Implementation
func (s *PostgresStore) GetByShift(shiftID int64) []*models.RosterEntry {
    rows, err := s.q.db.QueryContext(context.Background(), "SELECT id, shift_id, radiologist_id, start_date, status FROM roster_entries WHERE shift_id = $1", shiftID)
    if err != nil {
        return nil // Or log error
    }
    defer rows.Close()

    var entries []*models.RosterEntry
    for rows.Next() {
        var e models.RosterEntry
        if err := rows.Scan(&e.ID, &e.ShiftID, &e.RadiologistID, &e.StartDate, &e.Status); err != nil {
            continue
        }
        entries = append(entries, &e)
    }
    return entries
}

// Rules Service Implementation
func (s *PostgresStore) GetActive() []*models.AssignmentRule {
    rows, err := s.q.db.QueryContext(context.Background(), "SELECT id, name, priority_order, action_type, condition_filters, enabled FROM assignment_rules WHERE enabled = true ORDER BY priority_order ASC")
    if err != nil {
        return nil
    }
    defer rows.Close()

    var rules []*models.AssignmentRule
    for rows.Next() {
        var r models.AssignmentRule
        var filterJSON []byte
        if err := rows.Scan(&r.ID, &r.Name, &r.PriorityOrder, &r.ActionType, &filterJSON, &r.Enabled); err != nil {
            continue
        }
        if len(filterJSON) > 0 {
            json.Unmarshal(filterJSON, &r.ConditionFilters)
        }
        rules = append(rules, &r)
    }
    return rules
}
