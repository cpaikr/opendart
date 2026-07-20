package guide

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	DriftReportSchemaVersion = 1
	DriftReportKind          = "opendart-guide-drift"
	MaximumDriftReportBytes  = 1 << 20
	MaximumDriftFindings     = 64
)

var driftIdentifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
var driftPath = regexp.MustCompile(`^/[A-Za-z0-9._~/-]+$`)
var driftLocation = regexp.MustCompile(`^#(?:/[A-Za-z0-9._~:$@%+{},-]+)*$`)

// DriftReport is the complete producer/notifier boundary. It deliberately
// omits source values and full comparison details.
type DriftReport struct {
	SchemaVersion int              `json:"schemaVersion"`
	Kind          string           `json:"kind"`
	Outcome       string           `json:"outcome"`
	ObservedAt    string           `json:"observedAt"`
	RequestBudget RequestBudget    `json:"requestBudget"`
	Comparison    *DriftComparison `json:"comparison,omitempty"`
	Failure       *DriftFailure    `json:"failure,omitempty"`
}

type DriftComparison struct {
	TotalChanges    int            `json:"totalChanges"`
	BreakingChanges int            `json:"breakingChanges"`
	Findings        []DriftFinding `json:"findings,omitempty"`
	Truncated       bool           `json:"truncated"`
}

type DriftFinding struct {
	Change             string `json:"change"`
	LogicalOperationID string `json:"logicalOperationId,omitempty"`
	OperationID        string `json:"operationId,omitempty"`
	Method             string `json:"method,omitempty"`
	Path               string `json:"path,omitempty"`
	Location           string `json:"location"`
}

type DriftFailure struct {
	Code  string `json:"code"`
	Stage string `json:"stage"`
}

type DriftError struct {
	Failure DriftFailure
	cause   error
}

func (e *DriftError) Error() string {
	return fmt.Sprintf("guide drift %s failed (%s)", e.Failure.Stage, e.Failure.Code)
}

func (e *DriftError) Unwrap() error { return e.cause }

// DecodeDriftReport applies the strict bounded contract used by the isolated
// notifier before producer output can influence an issue.
func DecodeDriftReport(reader io.Reader) (DriftReport, error) {
	content, err := io.ReadAll(io.LimitReader(reader, MaximumDriftReportBytes+1))
	if err != nil {
		return DriftReport{}, errors.New("read guide drift report")
	}
	if len(content) == 0 || len(content) > MaximumDriftReportBytes {
		return DriftReport{}, errors.New("guide drift report size is invalid")
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	var report DriftReport
	if err := decoder.Decode(&report); err != nil {
		return DriftReport{}, errors.New("decode guide drift report")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return DriftReport{}, errors.New("guide drift report has trailing content")
	}
	if err := validateDriftReport(report); err != nil {
		return DriftReport{}, errors.New("validate guide drift report")
	}
	return report, nil
}

func validateDriftReport(report DriftReport) error {
	if report.SchemaVersion != DriftReportSchemaVersion || report.Kind != DriftReportKind {
		return errors.New("report identity is invalid")
	}
	if _, err := time.Parse("2006-01-02T15:04:05.000Z", report.ObservedAt); err != nil {
		return errors.New("report timestamp is invalid")
	}
	if report.RequestBudget.Ceiling < 0 || report.RequestBudget.Ceiling > AbsoluteDriftRequestLimit || report.RequestBudget.Used < 0 || report.RequestBudget.Used > report.RequestBudget.Ceiling {
		return errors.New("request budget is invalid")
	}
	switch report.Outcome {
	case "unchanged":
		if report.RequestBudget.Ceiling < len(Groups) || report.RequestBudget.Used != report.RequestBudget.Ceiling || report.Failure != nil || report.Comparison == nil || report.Comparison.TotalChanges != 0 || report.Comparison.BreakingChanges != 0 || len(report.Comparison.Findings) != 0 || report.Comparison.Truncated {
			return errors.New("unchanged report is inconsistent")
		}
	case "changed":
		if report.RequestBudget.Ceiling < len(Groups) || report.RequestBudget.Used != report.RequestBudget.Ceiling || report.Failure != nil || report.Comparison == nil || report.Comparison.TotalChanges <= 0 {
			return errors.New("changed report is inconsistent")
		}
	case "error":
		if report.Comparison != nil || report.Failure == nil || !allowedDriftFailure(*report.Failure) {
			return errors.New("error report is inconsistent")
		}
	default:
		return errors.New("report outcome is invalid")
	}
	if report.Comparison != nil {
		comparison := report.Comparison
		if comparison.BreakingChanges < 0 || len(comparison.Findings) > MaximumDriftFindings || len(comparison.Findings) > comparison.TotalChanges || comparison.Truncated != (len(comparison.Findings) < comparison.TotalChanges) {
			return errors.New("comparison aggregate is invalid")
		}
		for index, finding := range comparison.Findings {
			if !validDriftFinding(finding) {
				return fmt.Errorf("finding %d is invalid", index)
			}
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil || len(encoded) == 0 || len(encoded) > MaximumDriftReportBytes {
		return errors.New("encoded report size is invalid")
	}
	return nil
}

func validDriftFinding(finding DriftFinding) bool {
	if finding.Change != "added" && finding.Change != "removed" && finding.Change != "modified" {
		return false
	}
	if len(finding.Location) == 0 || len(finding.Location) > 512 || !driftLocation.MatchString(finding.Location) {
		return false
	}
	coordinates := finding.LogicalOperationID != "" || finding.OperationID != "" || finding.Method != "" || finding.Path != ""
	if !coordinates {
		return true
	}
	locationPath, ok := driftLocationPath(finding.Location)
	return ok && locationPath == finding.Path && driftIdentifier.MatchString(finding.LogicalOperationID) && driftIdentifier.MatchString(finding.OperationID) && finding.Method == http.MethodGet && driftPath.MatchString(finding.Path)
}

func allowedDriftFailure(failure DriftFailure) bool {
	expected := map[string]string{
		"acquisition-failed": "acquisition",
		"generation-failed":  "generation",
		"validation-failed":  "validation",
		"comparison-failed":  "comparison",
		"report-failed":      "report",
	}
	return expected[failure.Code] == failure.Stage
}
