# Missing End-to-End (E2E) Tests

The following features have been identified as lacking full E2E test coverage in `cmd/api/e2e_test.go`:

1.  **Assignment Rules Management (CRUD)**
    *   **Feature:** Creating, editing, and deleting assignment rules via the UI (`/rules`).
    *   **Status:** Backend handlers and Frontend UI exist, but the specific E2E tests (`TestE2ERules`) were replaced by Shift tests.
    *   **Missing Tests:** `CreateRule`, `EditRule`, `DeleteRule`.

2.  **Radiologist Management**
    *   **Feature:** Creating, editing, and deleting radiologists.
    *   **Status:** Currently strictly hardcoded in `cmd/api/main.go`. No UI or API endpoints exist.
    *   **Missing Tests:** `CreateRadiologist`, `EditRadiologist`, `DeleteRadiologist` (feature itself is missing).

3.  **Dashboard Statistics Verification**
    *   **Feature:** Real-time updates of "Assignments Today", "Active Radiologists", and "Pending Studies" on the Dashboard.
    *   **Status:** UI exists (`/`), but tests do not verify that these numbers update accurately after simulated assignments.
    *   **Missing Tests:** `VerifyDashboardStatsUpdate`.

4.  **SLA Escalation Visualization**
    *   **Feature:** Visual indication of escalated studies (e.g., color coding, badges) in the "Recent Assignments" list.
    *   **Status:** Backend logic exists (`ActionType: "ESCALATE"`), but UI visualization and E2E verification are missing.
    *   **Missing Tests:** `VerifyEscalationUI`.

5.  **Simulated Study Ingest**
    *   **Feature:** End-to-end flow from receiving an HL7 message (simulated) to visualization in the UI.
    *   **Status:** Currently tested via direct API call to `/api/simulate`. A true E2E would simulate the listener component.
    *   **Missing Tests:** `TestHL7IngestToDashboard`.
