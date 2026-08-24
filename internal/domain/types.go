package domain

import "time"

type DossierState string

const (
	StateDraft            DossierState = "DRAFT"
	StateSurveying        DossierState = "SURVEYING"
	StateAssessed         DossierState = "ASSESSED"
	StatePlanSubmitted    DossierState = "PLAN_SUBMITTED"
	StateChangesRequested DossierState = "CHANGES_REQUESTED"
	StateApproved         DossierState = "APPROVED"
	StateFrozen           DossierState = "FROZEN"
	StateReleased         DossierState = "RELEASED"
)

type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

func SeverityRank(s Severity) int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	}
	return 0
}

type FindingStatus string

const (
	FindingOpen     FindingStatus = "OPEN"
	FindingResolved FindingStatus = "RESOLVED"
)

type FindingSource string

const (
	SourceAutomatic FindingSource = "AUTOMATIC"
	SourceReview    FindingSource = "REVIEW"
)

type ReviewDecision string

const (
	DecisionPending ReviewDecision = "PENDING"
	DecisionReturn  ReviewDecision = "RETURN"
	DecisionApprove ReviewDecision = "APPROVE"
)

type SurveyDossier struct {
	ID             string                            `json:"id"`
	BuildingCode   string                            `json:"buildingCode"`
	Title          string                            `json:"title"`
	SurveyBoundary string                            `json:"surveyBoundary"`
	State          DossierState                      `json:"state"`
	Version        uint64                            `json:"version"`
	CreatedAt      time.Time                         `json:"createdAt"`
	UpdatedAt      time.Time                         `json:"updatedAt"`
	Components     map[string]TimberComponent        `json:"components"`
	Observations   map[string][]ConditionObservation `json:"observations"`
	Findings       map[string]ReviewFinding          `json:"findings"`
	Plans          []RepairPlanRevision              `json:"plans"`
	Manifest       *FreezeManifest                   `json:"manifest,omitempty"`
	Certificate    *WorkReleaseCertificate           `json:"certificate,omitempty"`
}
type TimberComponent struct {
	ID                 string   `json:"id"`
	DossierID          string   `json:"dossierID"`
	ComponentCode      string   `json:"componentCode"`
	ComponentType      string   `json:"componentType"`
	Location           string   `json:"location"`
	LoadPathParentCode string   `json:"loadPathParentCode"`
	RequiredChecks     []string `json:"requiredChecks"`
	BaselineNote       string   `json:"baselineNote"`
}
type ConditionObservation struct {
	ID             string             `json:"id"`
	DossierID      string             `json:"dossierID"`
	ComponentID    string             `json:"componentID"`
	Revision       uint32             `json:"revision"`
	ConditionType  string             `json:"conditionType"`
	LocationDetail string             `json:"locationDetail"`
	Severity       Severity           `json:"severity"`
	Measurements   map[string]float64 `json:"measurements"`
	EvidenceRefs   []string           `json:"evidenceRefs"`
	ObservedAt     time.Time          `json:"observedAt"`
	SupersedesID   string             `json:"supersedesID,omitempty"`
}
type ReviewFinding struct {
	ID                 string        `json:"id"`
	DossierID          string        `json:"dossierID"`
	Source             FindingSource `json:"source"`
	Code               string        `json:"code"`
	ComponentIDs       []string      `json:"componentIDs"`
	Severity           Severity      `json:"severity"`
	Message            string        `json:"message"`
	Blocking           bool          `json:"blocking"`
	Status             FindingStatus `json:"status"`
	ResolvedByRevision uint32        `json:"resolvedByRevision"`
}

type FindingChange string

const (
	FindingAdded     FindingChange = "ADDED"
	FindingUnchanged FindingChange = "UNCHANGED"
	FindingRemoved   FindingChange = "REMOVED"
)

type FindingDelta struct {
	Change  FindingChange `json:"change"`
	Finding ReviewFinding `json:"finding"`
}
type RepairAction struct {
	FindingID          string `json:"findingID"`
	ComponentID        string `json:"componentID"`
	Action             string `json:"action"`
	MaterialConstraint string `json:"materialConstraint"`
	AcceptanceStandard string `json:"acceptanceStandard"`
}
type RepairPlanRevision struct {
	ID                       string         `json:"id"`
	DossierID                string         `json:"dossierID"`
	Revision                 uint32         `json:"revision"`
	Actions                  []RepairAction `json:"actions"`
	ReferencedObservationIDs []string       `json:"referencedObservationIDs"`
	ResolvedFindingIDs       []string       `json:"resolvedFindingIDs"`
	MaterialConstraints      []string       `json:"materialConstraints"`
	AcceptanceCriteria       []string       `json:"acceptanceCriteria"`
	Decision                 ReviewDecision `json:"decision"`
	ReviewComments           []string       `json:"reviewComments"`
	SubmittedAt              time.Time      `json:"submittedAt"`
}
type FreezeManifest struct {
	DossierID      string    `json:"dossierID"`
	DossierVersion uint64    `json:"dossierVersion"`
	ComponentIDs   []string  `json:"componentIDs"`
	ObservationIDs []string  `json:"observationIDs"`
	PlanID         string    `json:"planID"`
	PlanRevision   uint32    `json:"planRevision"`
	ManifestHash   string    `json:"manifestHash"`
	FrozenAt       time.Time `json:"frozenAt"`
}
type WorkReleaseCertificate struct {
	ID                 string    `json:"id"`
	DossierID          string    `json:"dossierID"`
	DossierVersion     uint64    `json:"dossierVersion"`
	FreezeManifestHash string    `json:"freezeManifestHash"`
	AuditHeadHash      string    `json:"auditHeadHash"`
	IssuedBy           string    `json:"issuedBy"`
	IssuedAt           time.Time `json:"issuedAt"`
	CertificateDigest  string    `json:"certificateDigest"`
}
type AuditEvent struct {
	Sequence     uint64       `json:"sequence"`
	EventID      string       `json:"eventID"`
	DossierID    string       `json:"dossierID"`
	EventType    string       `json:"eventType"`
	Actor        string       `json:"actor"`
	State        DossierState `json:"state,omitempty"`
	OccurredAt   time.Time    `json:"occurredAt"`
	PayloadHash  string       `json:"payloadHash"`
	PreviousHash string       `json:"previousHash"`
	Hash         string       `json:"hash"`
}
type Timeline struct {
	Events         []AuditEvent `json:"events"`
	CurrentVersion uint64       `json:"currentVersion"`
	State          DossierState `json:"state"`
}
