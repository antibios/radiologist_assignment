Technical Infrastructure Document
Radiology Order Assignment Engine
Version: 1.0
Date: February 2026
Technology Stack: Go, sqlc, PostgreSQL, HL7, BeerCSS
Status: Draft

1. System Overview
1.1 Architecture Diagram
┌─────────────────────────────────────────────────────────────────────┐
│                     RIS / PACS Systems                               │
└────────────┬────────────────────────────────────────────────────────┘
             │
             │ HL7 ORM (Order Message)
             ▼
┌─────────────────────────────────────────────────────────────────────┐
│  HL7 Listener Service (Go)                                           │
│  - Parse inbound HL7 ORM messages                                    │
│  - Extract study metadata (modality, body_part, urgency, site)      │
│  - Enqueue to assignment queue                                       │
└────────────┬────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Assignment Engine (Go)                                              │
│  - Shift matching                                                    │
│  - Roster resolution                                                 │
│  - Rule evaluation pipeline                                          │
│  - Capacity/competency filtering                                     │
│  - SLA evaluation                                                    │
└────────────┬────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────┐
│  PostgreSQL Database (sqlc)                                          │
│  - Shifts, Rosters, Rules, Assignments                              │
│  - Radiologist credentials and availability                          │
│  - Audit logs                                                        │
└────────────┬────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────┐
│  HL7 Outbound Service (Go)                                           │
│  - Enrich inbound ORM with assigned radiologist                     │
│  - Emit HL7 ORU or OBR segment updates                              │
│  - Send to RIS / downstream systems                                  │
└────────────┬────────────────────────────────────────────────────────┘
             │
             ▼ HL7 ORU / OBR (with assigned radiologist)
┌─────────────────────────────────────────────────────────────────────┐
│  RIS / PACS / Worklist Systems                                       │
└─────────────────────────────────────────────────────────────────────┘
1.2 Key Components
ComponentPurposeTechnologyHL7 ListenerInbound message ingestion and parsingGo, hl7 libraryAssignment EngineCore logic for radiologist routingGo, in-memory cacheDatabase LayerPersistent storage and queriesPostgreSQL, sqlcHL7 EmitterOutbound message enrichmentGo, hl7 libraryWeb UIRule management, monitoring, reportingGo HTTP server, BeerCSSEvent QueueAsync work distributionNATS or Redis

2. HL7 Message Flow
2.1 Inbound HL7 ORM Message Example
MSH|^~\&|PACS|Robina|ASSIGNMENT_ENGINE|ENTERPRISE|20260204120530||ORM^O01|||2.5.1
PID|1||MRN123^^^Robina||Doe^John||19800115|M
OBR|1|STUDY_ID_123|ACCESSION_456|MRI MSK^MRI MUSCULOSKELETAL|||20260204120000|||^^^MSK|
OBX|1|TX|Indication^Clinical Indication||Patient with knee pain
ZRD|1|Robina^Site Code|MRI^Modality|STAT^Urgency
Key segments used for assignment:

MSH-3/4: Originating site (Robina)
OBR-4: Procedure code (MRI MSK)
OBX: Clinical context and indication
ZRD: Custom segment with modality, urgency, body part

2.2 Outbound HL7 ORU Message (Enriched)
MSH|^~\&|ASSIGNMENT_ENGINE|ENTERPRISE|PACS|Robina|20260204120531||ORU^R01|||2.5.1
PID|1||MRN123^^^Robina||Doe^John||19800115|M
OBR|1|STUDY_ID_123|ACCESSION_456|MRI MSK^MRI MUSCULOSKELETAL|||20260204120000|||^^^MSK|
OBX|1|TX|Indication^Clinical Indication||Patient with knee pain
ZRD|1|Robina^Site Code|MRI^Modality|STAT^Urgency
ZRA|1|DR_SMITH_ID^Smith^John^DR|MRI_MSK_ROBINA^Shift Name|LOAD_BALANCED^Assignment Strategy
New assignment segment (ZRA):

ZRA-2: Assigned radiologist (ID, name, credentials)
ZRA-3: Shift assignment
ZRA-4: Strategy used (load-balanced, urgent, escalated)
ZRA-5: Timestamp of assignment decision


3. Go Application Architecture
3.1 Project Structure
radiology-assignment/
├── cmd/
│   ├── listener/
│   │   └── main.go              # HL7 listener service
│   ├── engine/
│   │   └── main.go              # Assignment engine service
│   ├── emitter/
│   │   └── main.go              # HL7 outbound enrichment
│   └── api/
│       └── main.go              # Web API + BeerCSS UI
├── internal/
│   ├── hl7/
│   │   ├── parser.go            # Parse inbound HL7
│   │   ├── builder.go           # Build outbound HL7
│   │   └── types.go             # HL7 segment structs
│   ├── assignment/
│   │   ├── engine.go            # Core assignment logic
│   │   ├── rules.go             # Rule evaluation
│   │   ├── shift.go             # Shift matching
│   │   └── escalation.go        # SLA escalation
│   ├── models/
│   │   ├── radiologist.go
│   │   ├── shift.go
│   │   ├── roster.go
│   │   └── study.go
│   ├── db/
│   │   ├── queries/             # sqlc query definitions
│   │   └── schema.sql           # Database schema
│   └── cache/
│       ├── roster_cache.go      # In-memory roster
│       └── rules_cache.go       # In-memory rules
├── ui/
│   ├── templates/               # HTML templates (BeerCSS)
│   ├── static/
│   │   └── css/
│   │       └── beercss.min.css
│   └── assets/
├── sqlc.yaml                    # sqlc configuration
├── Dockerfile
├── docker-compose.yml
└── go.mod
3.2 Main Services
3.2.1 HL7 Listener Service (cmd/listener/main.go)
gopackage main

import (
    "context"
    "log"
    "net"
    
    "github.com/ehrlich/hl7"
    "your-module/internal/hl7"
    "your-module/internal/models"
    // ... other imports
)

func main() {
    listener, err := net.Listen("tcp", ":2575")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    defer listener.Close()
    
    log.Println("HL7 Listener started on :2575")
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Accept error: %v", err)
            continue
        }
        
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()
    
    // Read HL7 message (starts with MSH, ends with \r)
    buf := make([]byte, 65536)
    n, err := conn.Read(buf)
    if err != nil {
        log.Printf("Read error: %v", err)
        return
    }
    
    msgStr := string(buf[:n])
    
    // Parse HL7 message
    parsedMsg, err := hl7.ParseMessage(msgStr)
    if err != nil {
        log.Printf("Parse error: %v", err)
        sendNAK(conn, "Error parsing HL7 message")
        return
    }
    
    // Extract study metadata
    study := hl7.ExtractStudyMetadata(parsedMsg)
    
    // Enqueue to assignment queue
    queue.Enqueue(study)
    
    // Send ACK
    sendACK(conn, parsedMsg)
}

func sendACK(conn net.Conn, msg hl7.Message) {
    // Build ACK message
    ack := hl7.BuildACK(msg.MessageControlID(), "AA", "Message received")
    conn.Write([]byte(ack + "\r"))
}
3.2.2 Assignment Engine Service (cmd/engine/main.go)
gopackage main

import (
    "context"
    "log"
    "time"
    
    "your-module/internal/assignment"
    "your-module/internal/cache"
    "your-module/internal/db"
    "your-module/internal/models"
)

func main() {
    // Initialize database
    pgConn := db.Connect()
    defer pgConn.Close()
    
    // Initialize caches
    rosterCache := cache.NewRosterCache(pgConn)
    rulesCache := cache.NewRulesCache(pgConn)
    
    // Refresh caches periodically
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        for range ticker.C {
            rosterCache.Refresh(context.Background())
            rulesCache.Refresh(context.Background())
        }
    }()
    
    // Initialize assignment engine
    engine := assignment.NewEngine(pgConn, rosterCache, rulesCache)
    
    // Process queue
    for study := range queue.Channel() {
        assignment, err := engine.Assign(context.Background(), study)
        if err != nil {
            log.Printf("Assignment error for study %s: %v", study.ID, err)
            // Escalate to manual queue
            continue
        }
        
        // Persist assignment
        db.SaveAssignment(context.Background(), pgConn, assignment)
        
        // Emit event for outbound HL7
        events.Publish("assignment.completed", assignment)
    }
}
3.2.3 Assignment Engine Core Logic (internal/assignment/engine.go)
gopackage assignment

import (
    "context"
    "fmt"
    "log"
    
    "your-module/internal/cache"
    "your-module/internal/db"
    "your-module/internal/models"
)

type Engine struct {
    db          db.DBTX
    roster      *cache.RosterCache
    rules       *cache.RulesCache
}

func NewEngine(db db.DBTX, roster *cache.RosterCache, rules *cache.RulesCache) *Engine {
    return &Engine{db, roster, rules}
}

func (e *Engine) Assign(ctx context.Context, study *models.Study) (*models.Assignment, error) {
    // Step 1: Match shifts based on study characteristics
    shifts := e.matchShifts(ctx, study)
    if len(shifts) == 0 {
        return nil, fmt.Errorf("no matching shifts for study %s", study.ID)
    }
    
    // Step 2: Resolve radiologists from roster for matched shifts
    radiologists := e.resolveRadiologists(ctx, shifts)
    if len(radiologists) == 0 {
        return nil, fmt.Errorf("no available radiologists for shifts")
    }
    
    // Step 3: Apply rule-based assignment pipeline
    candidate := e.evaluateRules(ctx, study, radiologists)
    if candidate == nil {
        return nil, fmt.Errorf("no candidate selected after rule evaluation")
    }
    
    // Step 4: Apply competency and credentialing filters
    if !e.meetsCompetency(ctx, study, candidate) {
        return nil, fmt.Errorf("candidate %s lacks required competency", candidate.ID)
    }
    
    // Step 5: Check capacity constraints
    if !e.hasCapacity(ctx, study, candidate) {
        return nil, fmt.Errorf("candidate %s at capacity", candidate.ID)
    }
    
    // Step 6: Check SLA and escalate if needed
    escalated := e.checkSLA(ctx, study)
    
    // Step 7: Create assignment record
    assignment := &models.Assignment{
        StudyID:        study.ID,
        RadiologyID:    candidate.ID,
        ShiftID:        shifts[0].ID,
        AssignedAt:     time.Now(),
        Escalated:      escalated,
        Strategy:       "load_balanced",
    }
    
    return assignment, nil
}

func (e *Engine) matchShifts(ctx context.Context, study *models.Study) []*models.Shift {
    // Query shifts where work_type matches study modality/body_part
    shifts, err := db.GetShiftsByWorkType(ctx, e.db, study.Modality, study.BodyPart, study.Site)
    if err != nil {
        log.Printf("Error matching shifts: %v", err)
        return nil
    }
    return shifts
}

func (e *Engine) resolveRadiologists(ctx context.Context, shifts []*models.Shift) []*models.Radiologist {
    radiologists := make(map[string]*models.Radiologist)
    
    for _, shift := range shifts {
        roster := e.roster.GetByShift(shift.ID)
        for _, entry := range roster {
            if _, exists := radiologists[entry.RadiologyID]; !exists {
                rad, _ := db.GetRadiologist(ctx, e.db, entry.RadiologyID)
                radiologists[entry.RadiologyID] = rad
            }
        }
    }
    
    result := make([]*models.Radiologist, 0, len(radiologists))
    for _, rad := range radiologists {
        result = append(result, rad)
    }
    return result
}

func (e *Engine) evaluateRules(ctx context.Context, study *models.Study, radiologists []*models.Radiologist) *models.Radiologist {
    rules := e.rules.GetActive()
    
    for _, rule := range rules {
        if rule.Matches(study) {
            // Apply action: load balance across matching radiologists
            return e.loadBalance(radiologists)
        }
    }
    
    // Default: return first available radiologist
    return radiologists[0]
}

func (e *Engine) loadBalance(radiologists []*models.Radiologist) *models.Radiologist {
    // Return radiologist with lowest current workload
    var selected *models.Radiologist
    minLoad := int64(9999)
    
    for _, rad := range radiologists {
        load := e.getCurrentWorkload(rad.ID)
        if load < minLoad {
            minLoad = load
            selected = rad
        }
    }
    
    return selected
}

// ... Additional helper methods: meetsCompetency, hasCapacity, checkSLA, etc.

4. Database Schema and sqlc
4.1 Database Schema (db/schema.sql)
sqlCREATE TABLE shifts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    work_type VARCHAR(100) NOT NULL,
    sites TEXT[] NOT NULL,
    priority_level INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE radiologists (
    id VARCHAR(50) PRIMARY KEY,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    credentials TEXT[],
    specialties TEXT[],
    max_concurrent_studies INTEGER DEFAULT 20,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE roster_assignments (
    id BIGSERIAL PRIMARY KEY,
    shift_id BIGINT NOT NULL REFERENCES shifts(id),
    radiologist_id VARCHAR(50) NOT NULL REFERENCES radiologists(id),
    start_date DATE NOT NULL,
    end_date DATE,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE assignment_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    priority_order INTEGER NOT NULL,
    condition_filters JSONB NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    action_target VARCHAR(255),
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE study_assignments (
    id BIGSERIAL PRIMARY KEY,
    study_id VARCHAR(100) NOT NULL UNIQUE,
    radiologist_id VARCHAR(50) NOT NULL REFERENCES radiologists(id),
    shift_id BIGINT NOT NULL REFERENCES shifts(id),
    assigned_at TIMESTAMP DEFAULT NOW(),
    escalated BOOLEAN DEFAULT FALSE,
    assignment_strategy VARCHAR(50),
    rule_matched_id BIGINT REFERENCES assignment_rules(id),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50),
    entity_id VARCHAR(100),
    action VARCHAR(50),
    old_values JSONB,
    new_values JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE procedures (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(255),
    modality VARCHAR(50),
    body_part VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_procedures_code ON procedures(code);
CREATE INDEX idx_roster_shift ON roster_assignments(shift_id);
CREATE INDEX idx_roster_radiologist ON roster_assignments(radiologist_id);
CREATE INDEX idx_study_assignments_study ON study_assignments(study_id);
CREATE INDEX idx_study_assignments_radiologist ON study_assignments(radiologist_id);
4.2 sqlc Configuration (sqlc.yaml)
yamlversion: "2"
sql:
  - engine: "postgresql"
    queries: "internal/db/queries"
    schema: "internal/db/schema.sql"
    gen:
      go:
        out: "internal/db/generated"
        package: "db"
        emit_all_struct_fields: true
        emit_prepared_queries: true
        emit_interface: true
4.3 Sample sqlc Query Files (internal/db/queries)
shifts.sql
sql-- name: GetShiftsByWorkType :many
SELECT * FROM shifts 
WHERE work_type = $1 
  AND (sites @> ARRAY[$2]::TEXT[] OR sites = ARRAY[]::TEXT[])
  AND priority_level >= 0
ORDER BY priority_level DESC;

-- name: GetShiftByID :one
SELECT * FROM shifts WHERE id = $1;

-- name: InsertShift :exec
INSERT INTO shifts (name, work_type, sites, priority_level) 
VALUES ($1, $2, $3, $4);

-- name: UpdateShift :exec
UPDATE shifts SET name = $2, work_type = $3, updated_at = NOW() 
WHERE id = $1;
roster.sql
sql-- name: GetRosterByShift :many
SELECT * FROM roster_assignments 
WHERE shift_id = $1 
  AND status = 'active'
  AND start_date <= CURRENT_DATE 
  AND (end_date IS NULL OR end_date >= CURRENT_DATE);

-- name: GetRosterByRadiologist :many
SELECT s.* FROM roster_assignments ra
JOIN shifts s ON ra.shift_id = s.id
WHERE ra.radiologist_id = $1 
  AND ra.status = 'active'
  AND ra.start_date <= CURRENT_DATE 
  AND (ra.end_date IS NULL OR ra.end_date >= CURRENT_DATE);

-- name: InsertRosterAssignment :exec
INSERT INTO roster_assignments (shift_id, radiologist_id, start_date, end_date, status) 
VALUES ($1, $2, $3, $4, $5);
assignments.sql
sql-- name: GetStudyAssignment :one
SELECT * FROM study_assignments WHERE study_id = $1;

-- name: InsertStudyAssignment :one
INSERT INTO study_assignments 
(study_id, radiologist_id, shift_id, assignment_strategy, rule_matched_id) 
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRadiologistCurrentWorkload :one
SELECT COUNT(*) AS active_studies FROM study_assignments 
WHERE radiologist_id = $1 
  AND assigned_at > NOW() - INTERVAL '1 day';

5. HL7 Parsing and Building (internal/hl7/)
5.1 HL7 Parser (internal/hl7/parser.go)
gopackage hl7

import (
    "fmt"
    "strings"
    
    "github.com/ehrlich/hl7"
    "your-module/internal/models"
)

func ExtractStudyMetadata(msg hl7.Message) *models.Study {
    study := &models.Study{
        ID:        getSegmentField(msg, "OBR", 0, 3), // Accession
        MessageID: getSegmentField(msg, "MSH", 0, 9), // Message control ID
        Site:      getSegmentField(msg, "MSH", 0, 3), // Sending facility
        Timestamp: getSegmentField(msg, "OBR", 0, 6), // Observation datetime
    }
    
    // Extract modality and body part from procedure code
    procCode := getSegmentField(msg, "OBR", 0, 4)
    study.Modality, study.BodyPart = parseProcedureCode(procCode)
    
    // Extract urgency from ZRD segment if present
    study.Urgency = getSegmentField(msg, "ZRD", 0, 3)
    if study.Urgency == "" {
        study.Urgency = "ROUTINE"
    }
    
    // Extract clinical indication from OBX
    study.Indication = getSegmentField(msg, "OBX", 0, 5)
    
    return study
}

func parseProcedureCode(code string) (modality, bodyPart string) {
    // Expected format: "MRI MSK" or "CT CHEST"
    parts := strings.Fields(code)
    if len(parts) >= 1 {
        modality = parts[0]
    }
    if len(parts) >= 2 {
        bodyPart = parts[1]
    }
    return
}

func getSegmentField(msg hl7.Message, segmentType string, segmentNum int, fieldNum int) string {
    segments := msg.Segments(segmentType)
    if len(segments) <= segmentNum {
        return ""
    }
    
    segment := segments[segmentNum]
    if len(segment) <= fieldNum {
        return ""
    }
    
    return segment[fieldNum].String()
}
5.2 HL7 Builder (internal/hl7/builder.go)
gopackage hl7

import (
    "fmt"
    "time"
    
    "your-module/internal/models"
)

func BuildOutboundMessage(originalMsg string, assignment *models.Assignment, radiologist *models.Radiologist) string {
    // Parse original message
    msg := parseMessage(originalMsg)
    
    // Add assignment segment (ZRA)
    zraSegment := buildZRASegment(assignment, radiologist)
    
    // Insert ZRA after ZRD
    enrichedMsg := insertSegment(msg, zraSegment, "ZRD")
    
    return enrichedMsg.String()
}

func buildZRASegment(assignment *models.Assignment, radiologist *models.Radiologist) string {
    // ZRA|sequence|AssignedRadiologist^Name|Shift|Strategy|Timestamp
    
    timestamp := time.Now().Format("20060102150405")
    
    segment := fmt.Sprintf(
        "ZRA|1|%s^%s|%s|%s|%s",
        radiologist.ID,
        radiologist.LastName + "^" + radiologist.FirstName,
        assignment.ShiftID,
        assignment.Strategy,
        timestamp,
    )
    
    return segment
}

func insertSegment(msg hl7.Message, segment string, afterSegment string) hl7.Message {
    // Insert custom segment after existing segment type
    // Implementation depends on hl7 library API
    return msg
}

6. Web UI with BeerCSS (ui/)
6.1 Main Layout Template (ui/templates/layout.html)
html<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Radiology Assignment Engine</title>
    <link rel="stylesheet" href="/static/css/beercss.min.css">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto; }
        .container { max-width: 1200px; margin: 0 auto; padding: 1rem; }
        .metric { background: var(--form-element-background-color); padding: 1rem; border-radius: 0.5rem; }
        .status-active { color: #4caf50; }
        .status-escalated { color: #ff9800; }
    </style>
</head>
<body>
    <nav>
        <ul>
            <li><strong>Assignment Engine</strong></li>
        </ul>
        <ul>
            <li><a href="/">Dashboard</a></li>
            <li><a href="/rules">Rules</a></li>
            <li><a href="/shifts">Shifts</a></li>
            <li><a href="/roster">Roster</a></li>
            <li><a href="/assignments">Assignments</a></li>
        </ul>
    </nav>

    <div class="container">
        {{ template "content" . }}
    </div>
</body>
</html>
6.2 Dashboard Template (ui/templates/dashboard.html)
html{{ define "dashboard" }}
<h1>Assignment Engine Dashboard</h1>

<div style="display: grid; grid-template-columns: repeat(4, 1fr); gap: 1rem; margin-bottom: 2rem;">
    <div class="metric">
        <h3>{{ .TodayAssignments }}</h3>
        <p>Assignments Today</p>
    </div>
    <div class="metric">
        <h3 class="status-active">{{ .SLACompliance }}%</h3>
        <p>SLA Compliance</p>
    </div>
    <div class="metric">
        <h3 class="status-escalated">{{ .Escalations }}</h3>
        <p>Escalations</p>
    </div>
    <div class="metric">
        <h3>{{ .AvgLatency }}ms</h3>
        <p>Avg Assignment Latency</p>
    </div>
</div>

<h2>Recent Assignments</h2>
<table>
    <thead>
        <tr>
            <th>Study ID</th>
            <th>Modality</th>
            <th>Assigned Radiologist</th>
            <th>Shift</th>
            <th>Assigned At</th>
            <th>Strategy</th>
            <th>Status</th>
        </tr>
    </thead>
    <tbody>
        {{ range .RecentAssignments }}
        <tr>
            <td>{{ .StudyID }}</td>
            <td>{{ .Modality }}</td>
            <td>{{ .RadiologistName }}</td>
            <td>{{ .ShiftName }}</td>
            <td>{{ .AssignedAt }}</td>
            <td>{{ .Strategy }}</td>
            <td>
                {{ if .Escalated }}
                    <span class="status-escalated">Escalated</span>
                {{ else }}
                    <span class="status-active">Assigned</span>
                {{ end }}
            </td>
        </tr>
        {{ end }}
    </tbody>
</table>

<h2>Active Radiologists & Workload</h2>
<table>
    <thead>
        <tr>
            <th>Radiologist</th>
            <th>Active Studies</th>
            <th>Capacity</th>
            <th>Shifts</th>
            <th>Status</th>
        </tr>
    </thead>
    <tbody>
        {{ range .RadiologistLoad }}
        <tr>
            <td>{{ .Name }}</td>
            <td>{{ .ActiveStudies }}</td>
            <td>{{ .UsagePercent }}%</td>
            <td>{{ .ShiftNames }}</td>
            <td>
                {{ if .AtCapacity }}
                    <span style="color: #f44336;">At Capacity</span>
                {{ else }}
                    <span class="status-active">Available</span>
                {{ end }}
            </td>
        </tr>
        {{ end }}
    </tbody>
</table>
{{ end }}
6.3 Procedure Management Template (ui/templates/procedures.html)
html{{ define "procedures" }}
<h1>Procedure Management</h1>
<button onclick="showAddProcedureModal()">+ Add Procedure</button>
<table>
    <thead>
        <tr>
            <th>Code</th>
            <th>Description</th>
            <th>Modality</th>
            <th>Body Part</th>
            <th>Actions</th>
        </tr>
    </thead>
    <tbody>
        {{ range .Procedures }}
        <tr>
            <td>{{ .Code }}</td>
            <td>{{ .Description }}</td>
            <td>{{ .Modality }}</td>
            <td>{{ .BodyPart }}</td>
            <td>
                <button onclick="editProcedure('{{ .Code }}')">Edit</button>
                <button onclick="deleteProcedure('{{ .Code }}')">Delete</button>
            </td>
        </tr>
        {{ end }}
    </tbody>
</table>
{{ end }}

6.4 Rules Management Template (ui/templates/rules.html)
html{{ define "rules" }}
<h1>Assignment Rules</h1>

<button onclick="showAddRuleModal()">+ Add Rule</button>

<table>
    <thead>
        <tr>
            <th>Priority</th>
            <th>Name</th>
            <th>Conditions</th>
            <th>Action</th>
            <th>Status</th>
            <th>Actions</th>
        </tr>
    </thead>
    <tbody>
        {{ range .Rules }}
        <tr>
            <td>{{ .Priority }}</td>
            <td>{{ .Name }}</td>
            <td><code>{{ .ConditionsPreview }}</code></td>
            <td>{{ .ActionType }} → {{ .ActionTarget }}</td>
            <td>
                {{ if .Enabled }}
                    <span class="status-active">Enabled</span>
                {{ else }}
                    <span style="color: #999;">Disabled</span>
                {{ end }}
            </td>
            <td>
                <button onclick="editRule({{ .ID }})">Edit</button>
                <button onclick="deleteRule({{ .ID }})">Delete</button>
            </td>
        </tr>
        {{ end }}
    </tbody>
</table>

<dialog id="ruleModal">
    <article>
        <h2>Add/Edit Rule</h2>
        <form onsubmit="saveRule(event)">
            <label>
                Rule Name
                <input type="text" id="ruleName" required>
            </label>
            <label>
                Priority Order
                <input type="number" id="rulePriority" required>
            </label>
            <label>
                Conditions (JSON)
                <textarea id="ruleConditions" rows="6" required></textarea>
            </label>
            <label>
                Action Type
                <select id="actionType">
                    <option>ASSIGN_TO_SHIFT</option>
                    <option>ASSIGN_TO_RADIOLOGIST</option>
                    <option>ESCALATE</option>
                </select>
            </label>
            <label>
                Action Target
                <input type="text" id="actionTarget">
            </label>
            <label>
                <input type="checkbox" id="ruleEnabled" checked>
                Enabled
            </label>
            <button type="submit">Save Rule</button>
            <button type="button" onclick="closeModal()">Cancel</button>
        </form>
    </article>
</dialog>
{{ end }}

7. In-Memory Caching (internal/cache/)
7.1 Roster Cache (internal/cache/roster_cache.go)
gopackage cache

import (
    "context"
    "log"
    "sync"
    "time"
    
    "your-module/internal/db"
    "your-module/internal/models"
)

type RosterCache struct {
    mu       sync.RWMutex
    data     map[string][]*models.RosterEntry // key: shift_id
    db       db.DBTX
    lastSync time.Time
}

func NewRosterCache(db db.DBTX) *RosterCache {
    rc := &RosterCache{
        data: make(map[string][]*models.RosterEntry),
        db:   db,
    }
    rc.Refresh(context.Background())
    return rc
}

func (rc *RosterCache) Refresh(ctx context.Context) error {
    // Query all active roster assignments
    entries, err := db.GetAllActiveRosterAssignments(ctx, rc.db)
    if err != nil {
        return err
    }
    
    rc.mu.Lock()
    defer rc.mu.Unlock()
    
    rc.data = make(map[string][]*models.RosterEntry)
    for _, entry := range entries {
        rc.data[entry.ShiftID] = append(rc.data[entry.ShiftID], entry)
    }
    
    rc.lastSync = time.Now()
    log.Printf("Roster cache refreshed with %d entries", len(entries))
    
    return nil
}

func (rc *RosterCache) GetByShift(shiftID string) []*models.RosterEntry {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    
    if entries, ok := rc.data[shiftID]; ok {
        return entries
    }
    return []*models.RosterEntry{}
}
7.2 Rules Cache (internal/cache/rules_cache.go)
gopackage cache

import (
    "context"
    "log"
    "sync"
    "time"
    
    "your-module/internal/db"
    "your-module/internal/models"
)

type RulesCache struct {
    mu       sync.RWMutex
    rules    []*models.AssignmentRule
    db       db.DBTX
    lastSync time.Time
}

func NewRulesCache(db db.DBTX) *RulesCache {
    rc := &RulesCache{db: db}
    rc.Refresh(context.Background())
    return rc
}

func (rc *RulesCache) Refresh(ctx context.Context) error {
    // Query all enabled rules ordered by priority
    rules, err := db.GetActiveRulesOrderedByPriority(ctx, rc.db)
    if err != nil {
        return err
    }
    
    rc.mu.Lock()
    defer rc.mu.Unlock()
    
    rc.rules = rules
    rc.lastSync = time.Now()
    log.Printf("Rules cache refreshed with %d rules", len(rules))
    
    return nil
}

func (rc *RulesCache) GetActive() []*models.AssignmentRule {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    
    return rc.rules
}

8. Deployment Architecture
8.1 Docker Compose (docker-compose.yml)
yamlversion: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: radiology
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: assignment_engine
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./internal/db/schema.sql:/docker-entrypoint-initdb.d/init.sql

  listener:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        SERVICE: listener
    ports:
      - "2575:2575"
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://radiology:secret@postgres:5432/assignment_engine
      QUEUE_URL: nats://nats:4222

  engine:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        SERVICE: engine
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://radiology:secret@postgres:5432/assignment_engine
      QUEUE_URL: nats://nats:4222

  emitter:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        SERVICE: emitter
    ports:
      - "2576:2576"
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://radiology:secret@postgres:5432/assignment_engine
      QUEUE_URL: nats://nats:4222

  api:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        SERVICE: api
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://radiology:secret@postgres:5432/assignment_engine

  nats:
    image: nats:latest
    ports:
      - "4222:4222"
      - "8222:8222"

volumes:
  postgres_data:
8.2 Dockerfile
dockerfileFROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/${SERVICE} ./cmd/${SERVICE}

FROM alpine:latest
RUN apk --no-cache add ca-certificates

ARG SERVICE
COPY --from=builder /bin/${SERVICE} /bin/service

EXPOSE 2575 2576 8080
CMD ["/bin/service"]

9. Message Flow Diagram
mermaidsequenceDiagram
    participant RIS as RIS/PACS
    participant Listener as HL7 Listener
    participant Queue as Event Queue
    participant Engine as Assignment Engine
    participant DB as PostgreSQL
    participant Cache as In-Memory Cache
    participant Emitter as HL7 Emitter
    
    RIS->>Listener: HL7 ORM (Study Order)
    Listener->>Listener: Parse HL7 Message
    Listener->>DB: Log inbound message
    Listener->>Queue: Enqueue StudyMetadata
    Listener->>RIS: Send ACK
    
    Engine->>Queue: Dequeue StudyMetadata
    Engine->>Cache: Get roster by shift
    Engine->>Cache: Get active rules
    Engine->>DB: Get shifts for modality/body_part
    Engine->>DB: Get radiologist credentials
    Engine->>Engine: Evaluate assignment pipeline
    Engine->>DB: Save assignment decision
    Engine->>Queue: Publish assignment.completed
    
    Emitter->>Queue: Subscribe assignment.completed
    Emitter->>DB: Get assignment details
    Emitter->>Emitter: Build outbound HL7 ORU
    Emitter->>Emitter: Enrich with ZRA segment
    Emitter->>RIS: Send HL7 ORU (assigned radiologist)
    
    RIS->>RIS: Update worklist with radiologist
    RIS->>RIS: Display study in assigned radiologist queue

10. Performance Considerations
10.1 Caching Strategy

Roster Cache: Refreshed every 5 minutes or on-demand via webhook
Rules Cache: Refreshed every 5 minutes
Radiologist Workload: In-memory counter updated on each assignment, synced to DB every 30 seconds
Shift Metadata: Pre-loaded at startup, refreshed hourly

10.2 Database Optimization

Indexes on: roster_assignments(shift_id, radiologist_id), study_assignments(radiologist_id), assignments(study_id)
Query optimization: Use prepared statements via sqlc
Connection pooling: pgx with 25-50 connections

10.3 HL7 Processing

Listener: Multi-threaded, handles up to 1000 concurrent connections
Queue: NATS for at-least-once delivery semantics
Batch processing: Assignment engine processes in batches of 100 studies


11. Monitoring and Observability
11.1 Metrics to Export (Prometheus)
assignment_engine_assignments_total{shift, strategy, escalated}
assignment_engine_assignment_latency_seconds{quantile}
assignment_engine_sla_breaches_total{shift, modality}
assignment_engine_rule_matches_total{rule_id}
hl7_listener_messages_received_total
hl7_emitter_messages_sent_total
roster_cache_refresh_seconds
rules_cache_refresh_seconds
11.2 Health Checks

GET /health: Overall system health
GET /health/db: Database connectivity
GET /health/queue: Message queue connectivity
GET /metrics: Prometheus metrics endpoint


12. Security Considerations

Database: Use connection strings from environment variables
HL7 Listener: Validate message structure before processing
API: Implement role-based access control (shift managers, admins)
Audit Logging: All rule changes and assignments logged with user context
Network: Run services in isolated VPC with network policies
