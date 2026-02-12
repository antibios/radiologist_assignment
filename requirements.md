Software Requirements Document
Radiology Order Assignment Engine
Version: 1.0
Date: February 2026
Status: Draft

1. Executive Summary
This document defines the requirements for a work-type centric radiology order assignment system designed to intelligently route diagnostic studies to appropriate radiologists based on shift definitions, clinical competencies, and dynamic roster configurations. The system must handle enterprise-scale operations (10,000+ studies daily across 150+ radiologists) with rule-based assignment logic and real-time SLA escalation capabilities.

2. Problem Statement
2.1 Current State Limitations
Existing site-centric or radiologist-centric shift models fail to address the operational reality of radiology departments where:

Site-Centric Model Failure: Individual sites cannot be the primary organizational unit because sites generate work beyond the capacity and competency of on-site radiologists. Work must be distributed to remote radiologists based on credentials, subspecialty, and availability.
Radiologist-Centric Model Failure: Daily roster changes would require continuous rebuilding of individual radiologist shifts. Each credential or availability change cascades into multiple shift modifications, creating an unsustainable maintenance burden.
Current Worklist Approach Inadequacy: Using MRQ worklist stacks as a solution creates an ever-moving target requiring unwieldy numbers of constantly changing assignment rules that cannot capture nuanced, context-dependent allocation decisions.

2.2 Operational Complexity at Scale

186+ clinical sites, each producing 3-5 distinct work types
Over 10,000 studies generated daily
150+ radiologists on roster at any given time
Radiologists scheduled across multiple shifts simultaneously
SLA escalation requirements for aging studies
Complex interdependencies requiring evaluation of 1,450+ user/shift combinations per assignment cycle


3. Goals and Objectives
3.1 Primary Objectives

Work-Type Centric Shift Definition: Define shifts as distinct types of clinical work (e.g., "MRI MSK Robina," "Urgent Cases Robina") rather than sites or individuals.
Roster-Driven Allocation: Use roster assignments to specify which radiologist is responsible for each shift, with each radiologist's worklist dynamically computed as the union of all assigned shifts.
Rule-Based Intelligent Routing: Implement assignment rules that route studies to appropriate radiologists based on shift responsibility, competency, capacity, and special arrangements.
Scalable Architecture: Support enterprise-level operations without requiring manual intervention for daily roster changes.
SLA Compliance: Implement escalation logic to ensure studies meet target completion times and escalate appropriately when thresholds are breached.


4. Functional Requirements
4.1 Shift Management
FR-4.1.1: The system shall define shifts as work types, each characterized by:

Unique shift identifier
Clinical work type description (e.g., modality, body part, urgency level)
Associated site(s)
Default priority level
Assigned radiologists (roster)

FR-4.1.2: The system shall support multiple shifts per site and allow shifts to span multiple sites.
FR-4.1.3: The system shall maintain audit trails for all shift configuration changes with effective dating.
4.2 Roster Management
FR-4.2.1: The system shall maintain daily roster data specifying which radiologists are assigned to which shifts.
FR-4.2.2: Roster changes shall take effect immediately without requiring shift rebuilds or manual intervention.
FR-4.2.3: The system shall compute each radiologist's effective worklist as the union of all shifts to which they are rostered.
FR-4.2.4: The system shall handle radiologists scheduled to multiple shifts simultaneously with clear visibility of shift responsibilities.
4.3 Assignment Rules Engine
FR-4.3.1: The system shall evaluate assignment rules in priority order, with at least the following rule categories supported:

Shift-based assignment (primary: route to radiologist(s) rostered for this shift)
Competency-based filters (subspecialty credentialing, modality-specific qualifications)
Capacity constraints (workload balancing, maximum concurrent studies)
Credentialing requirements (state licensure, facility-specific credentials)
Special arrangement overrides (bespoke allocations, preferred radiologist assignments)
Overflow rules (fallback radiologists when primary assignment unavailable)
Cross-site enterprise rules (e.g., ILO arrangements transcending shift definitions)

FR-4.3.2: The system shall support at least 1,000 active assignment rules.
FR-4.3.3: The system shall allow rules to be evaluated against multiple study attributes including modality, body part, urgency, clinical indication, originating site, and study age.
4.4 Study Assignment
FR-4.4.1: For each inbound study, the system shall:

Extract study metadata (modality, body part, urgency, site, etc.)
Identify applicable shifts based on study characteristics
Identify radiologists rostered to those shifts
Apply competency and credentialing filters
Apply capacity constraints
Route to optimal radiologist or invoke escalation logic

FR-4.4.2: The system shall assign studies in real-time (sub-second latency) with no batch delays.
FR-4.4.3: The system shall support assignment to a specific radiologist or to a worklist managed by multiple radiologists.
4.5 SLA and Escalation Management
FR-4.5.1: The system shall track study age from ingest timestamp and compare against defined SLA thresholds by study type.
FR-4.5.2: When a study exceeds SLA threshold, the system shall:

Flag the study for escalation
Reassign to available radiologists with higher priority
Notify supervisors per escalation policy
Invoke overflow rules if primary assignment unavailable

FR-4.5.3: The system shall support tiered escalation (e.g., 15-minute soft alert, 30-minute hard reassignment, 60-minute supervisor notification).
4.6 Availability and Capacity Management
FR-4.6.1: The system shall track radiologist availability status (on-shift, available, at-capacity, offline).
FR-4.6.2: The system shall implement load-balancing logic to distribute work evenly across available radiologists on a shift.
FR-4.6.3: The system shall respect maximum concurrent study limits per radiologist if configured.
4.7 Procedure Management
FR-4.7.1: The system shall maintain a catalog of clinical procedures, each defined by a unique code and description.
FR-4.7.2: The system shall support mapping each procedure code to a specific modality and body part to facilitate accurate normalization of inbound study data.
FR-4.7.3: The system shall provide an interface for authorized users to create, update, and delete procedure definitions.

5. Non-Functional Requirements
5.1 Performance
NFR-5.1.1: Assignment decision latency shall not exceed 500ms (p95) from study submission to assignment.
NFR-5.1.2: The system shall support 10,000+ studies per day with consistent performance.
NFR-5.1.3: Rule evaluation shall complete within acceptable latency across 1,000+ rules.
5.2 Scalability
NFR-5.2.1: The system shall scale to support 200+ radiologists without architectural changes.
NFR-5.2.2: The system shall scale to support 500+ shifts without performance degradation.
NFR-5.2.3: The system shall evaluate up to 1,500 user/shift combinations per assignment cycle.
5.3 Availability
NFR-5.3.1: The system shall maintain 99.5% uptime during operational hours.
NFR-5.3.2: The system shall implement graceful degradation if assignment rules cannot be evaluated (fallback to default rules).
NFR-5.3.3: Assignment failures shall be logged and escalated for manual intervention.
5.4 Maintainability
NFR-5.4.1: Rule changes shall take effect without system restart.
NFR-5.4.2: Roster changes shall propagate within 30 seconds.
NFR-5.4.3: The system shall provide audit logging of all assignment decisions for compliance and troubleshooting.
5.5 Auditability and Compliance
NFR-5.5.1: The system shall log assignment decisions including rule matched, radiologist assigned, timestamp, and rationale.
NFR-5.5.2: The system shall maintain immutable audit trails for all shifts, rosters, and rule changes.
NFR-5.5.3: The system shall support compliance reports showing assignment distribution, SLA compliance, and escalation frequency.

6. System Architecture Concepts
6.1 Data Model
Shifts Table

shift_id (unique identifier)
shift_name (human-readable description, e.g., "MRI MSK Robina")
work_type (modality/clinical category)
site_ids (array of sites)
priority_level (default priority)
created_at, updated_at

Roster Table

roster_id (unique identifier)
shift_id
radiologist_id
start_date, end_date
status (active/inactive)

Assignment Rules Table

rule_id (unique identifier)
rule_name
priority_order
condition_filters (JSON: modality, body_part, urgency, site, etc.)
action (assignment target: shift, radiologist, worklist)
created_at, updated_at

Studies/Orders Table

study_id
modality, body_part, urgency
originating_site_id
assigned_radiologist_id
assigned_shift_id
assignment_timestamp
sla_threshold, sla_status
escalation_history

6.2 Assignment Engine Architecture
The assignment engine shall operate as follows:

Study Ingest: Receive study metadata and normalize attributes
Shift Matching: Identify shifts responsible for this work type
Radiologist Resolution: Query roster for radiologists assigned to identified shifts
Filter Pipeline: Apply competency, credentialing, and capacity filters
Rule Evaluation: Iterate through assignment rules in priority order, stopping at first match
Load Balancing: Distribute across multiple radiologists if applicable
Escalation Check: Evaluate against SLA thresholds
Assignment: Route to determined radiologist(s) and persist assignment record

6.3 Real-Time Roster Synchronization
The system shall maintain an in-memory representation of the current roster for sub-second lookup during assignment. Updates shall be propagated via:

Scheduled roster refresh (every 5 minutes)
Event-driven updates when roster changes occur
Eventual consistency model with reconciliation


7. Assignment Rules Examples
Rule 1: Shift-Based Primary Assignment

Condition: Study modality in [MRI], body_part in [MSK], site = Robina
Action: Assign to any radiologist rostered to "MRI MSK Robina" shift (load-balanced)

Rule 2: Competency Filter

Condition: Study modality = [CT], site = Metro
Action: Filter Rule 1 result to only radiologists with CT subspecialty credential

Rule 3: Special Arrangement

Condition: Study urgency = STAT, site = Remote
Action: Assign to "Dr. Smith" (preferred STAT radiologist)

Rule 4: Overflow

Condition: All primary radiologists at capacity
Action: Route to "Overflow" shift (radiologists with flexible schedules)

Rule 5: SLA Escalation

Condition: Study age > 30 minutes AND not assigned
Action: Escalate to supervisor AND assign to next available radiologist regardless of shift

Rule 6: Enterprise-Level (ILO)

Condition: Study origin = Partner Network
Action: Assign per inter-library loan agreement (specific radiologist pool)


8. Integration Points
FR-8.1: The system shall integrate with the PACS/RIS to receive study metadata and order details.
FR-8.2: The system shall integrate with the roster/scheduling system to consume current shift assignments.
FR-8.3: The system shall integrate with the credential/compliance system to validate radiologist qualifications.
FR-8.4: The system shall expose APIs for:

Querying current radiologist assignments and worklist composition
Updating shift definitions and roster assignments
Retrieving assignment history and audit logs
Triggering manual reassignment if needed


9. Success Criteria

System successfully routes 10,000+ daily studies with <1% unassigned studies
95% of studies assigned within 500ms
SLA escalation rules execute automatically with <5% manual intervention
Roster changes take effect within 30 seconds
No performance degradation as number of shifts, rules, or radiologists increases
Complete audit trail maintained for compliance reporting


10. Future Enhancements (Out of Scope)

Machine learning-based radiologist matching
Fatigue/workload-aware assignment
Predictive SLA breach prevention
Dynamic shift optimization
Multi-enterprise federation scenarios
