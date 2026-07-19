// Package liveconformance owns the complete, explicitly invoked OpenDART live
// operation matrix. It treats the committed OpenAPI document as routing and
// structural policy while typed cases own stable inputs and semantic checks.
package liveconformance

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

const (
	ReportSchemaVersion  = 2
	ReportKind           = "opendart-live-conformance"
	TrustedServer        = "https://opendart.fss.or.kr/api"
	CredentialSource     = "OPENDART_API_KEY"
	MaximumBodyBytes     = 8 << 20
	MaximumArchiveBytes  = 64 << 20
	AbsoluteRequestLimit = 200
	RequestTimeout       = 30 * time.Second
	RequestPacing        = 100 * time.Millisecond
)

type AssertionID string

type Case struct {
	ID             string
	Method         string
	Path           string
	Representation string
	Parameters     map[string][]string
	Assertion      AssertionID
	Discovery      DiscoveryID
}

type DiscoveryID string

// Discovery declares a fixed, bounded set of already-valid OpenAPI requests
// whose results may supply coordinates to multiple primary cases.
type Discovery struct {
	ID          DiscoveryID
	MaxRequests int
	Requests    []DiscoveryRequest
	Targets     []DiscoveryTarget
}

type DiscoveryRequest struct {
	ID         string
	Parameters map[string][]string
}

type DiscoveryTarget struct {
	CaseID      string
	DetailTypes []string
	Aliases     []string
}

func (c Case) operationIdentity() string {
	return c.Method + " " + c.Path + " " + c.Representation
}

type Assertion struct {
	Representations []string
	Check           func(Response) (ComparisonEvidence, bool)
}

type Response struct {
	Representation   string
	APIStatus        string
	JSON             map[string]any
	XMLValues        map[string][]string
	Archive          ArchiveSummary
	ArchiveDocuments []ArchiveDocument
}

type ArchiveSummary struct {
	Entries       int `json:"entries"`
	XMLDocuments  int `json:"xmlDocuments"`
	ExpandedBytes int `json:"expandedBytes"`
}

type ArchiveDocument struct {
	Root      string
	XMLValues map[string][]string
}

type ComparisonEvidence struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

type RequestBudget struct {
	Ceiling          int `json:"ceiling"`
	Used             int `json:"used"`
	DiscoveryCeiling int `json:"discoveryCeiling"`
	DiscoveryUsed    int `json:"discoveryUsed"`
}

type Report struct {
	SchemaVersion       int           `json:"schemaVersion"`
	Kind                string        `json:"kind"`
	Outcome             string        `json:"outcome"`
	ObservedAt          string        `json:"observedAt"`
	CredentialSource    string        `json:"credentialSource"`
	CredentialPersisted bool          `json:"credentialPersisted"`
	RequestBudget       RequestBudget `json:"requestBudget"`
	Cases               []CaseResult  `json:"cases"`
	Failure             *Failure      `json:"failure,omitempty"`
}

type CaseResult struct {
	CaseID             string             `json:"caseId"`
	OperationID        string             `json:"operationId"`
	LogicalOperationID string             `json:"logicalOperationId"`
	Method             string             `json:"method"`
	Path               string             `json:"path"`
	Representation     string             `json:"representation"`
	AssertionID        AssertionID        `json:"assertionId"`
	Outcome            string             `json:"outcome"`
	HTTPStatus         int                `json:"httpStatus"`
	MediaType          string             `json:"mediaType,omitempty"`
	APIStatus          string             `json:"apiStatus,omitempty"`
	BodyBytes          int                `json:"bodyBytes"`
	BodySHA256         string             `json:"bodySha256,omitempty"`
	SchemaLocation     string             `json:"schemaLocation"`
	Comparison         ComparisonEvidence `json:"comparison"`
}

type Failure struct {
	Code        string      `json:"code"`
	Stage       string      `json:"stage"`
	CaseID      string      `json:"caseId,omitempty"`
	DiscoveryID DiscoveryID `json:"discoveryId,omitempty"`
	Operation   string      `json:"operation,omitempty"`
}

type Error struct {
	Failure Failure
}

func (e *Error) Error() string {
	if e.Failure.CaseID == "" {
		return fmt.Sprintf("live conformance %s failed (%s)", e.Failure.Stage, e.Failure.Code)
	}
	return fmt.Sprintf("live conformance %s failed for case %s (%s)", e.Failure.Stage, e.Failure.CaseID, e.Failure.Code)
}

type specification interface {
	Operations() (openapispec.OperationCatalog, error)
	ValidateRequest(method, path string, query url.Values) error
	ValidateResponse(method, path, contentType string, status int, body []byte) error
}

type preparedCase struct {
	definition          Case
	operation           openapispec.Operation
	query               url.Values
	assertion           Assertion
	allowEmptyDiscovery bool
}

// Plan is returned only after every operation, request, trust, assertion, and
// budget invariant has passed without reading a credential or using a network.
type Plan struct {
	specification   specification
	cases           []preparedCase
	discoveries     []preparedDiscovery
	requestBudget   int
	discoveryBudget int
}

type preparedDiscovery struct {
	definition Discovery
	requests   []preparedCase
}

type dependencies struct {
	do         func(*http.Request) (*http.Response, error)
	credential func() (string, error)
	now        func() time.Time
	wait       func(time.Duration) error
}
