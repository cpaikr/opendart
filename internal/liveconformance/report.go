package liveconformance

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const MaximumReportBytes = 1 << 20

var safeIdentifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
var safePath = regexp.MustCompile(`^/[A-Za-z0-9._~/-]+$`)
var apiStatus = regexp.MustCompile(`^[0-9]{3}$`)

// DecodeReport applies the same bounded, strict, allowlisted contract that an
// isolated notifier uses before treating producer output as data.
func DecodeReport(reader io.Reader) (Report, error) {
	limited := io.LimitReader(reader, MaximumReportBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return Report{}, errors.New("read live conformance report")
	}
	if len(content) == 0 || len(content) > MaximumReportBytes {
		return Report{}, errors.New("live conformance report size is invalid")
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	var report Report
	if err := decoder.Decode(&report); err != nil {
		return Report{}, errors.New("decode live conformance report")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Report{}, errors.New("live conformance report has trailing content")
	}
	if err := validateReport(report, ""); err != nil {
		return Report{}, errors.New("validate live conformance report")
	}
	return report, nil
}

func validateAllowlistedFields(report Report) error {
	seenCases := make(map[string]bool, len(report.Cases))
	caseOperations := make(map[string]string, len(report.Cases))
	for index, result := range report.Cases {
		if !safeIdentifier.MatchString(result.CaseID) || !safeIdentifier.MatchString(result.OperationID) || !safeIdentifier.MatchString(result.LogicalOperationID) || seenCases[result.CaseID] {
			return fmt.Errorf("live conformance report case %d has an invalid identity", index)
		}
		seenCases[result.CaseID] = true
		caseOperations[result.CaseID] = result.Method + " " + result.Path + " " + result.Representation
		if result.Method != http.MethodGet || !validOperationPath(result.Path) || !allowedRepresentation(result.Representation) || !safeIdentifier.MatchString(string(result.AssertionID)) {
			return fmt.Errorf("live conformance report case %d has invalid request coordinates", index)
		}
		if result.Outcome != "passed" && result.Outcome != "failed" {
			return fmt.Errorf("live conformance report case %d has invalid outcome", index)
		}
		expectedSchemaLocation := result.Path + "#responses/default/content/" + escapePointer(result.Representation)
		if result.HTTPStatus != 0 && (result.HTTPStatus < 100 || result.HTTPStatus > 599) || result.BodyBytes < 0 || result.BodyBytes > MaximumBodyBytes || result.APIStatus != "" && !apiStatus.MatchString(result.APIStatus) || result.SchemaLocation != expectedSchemaLocation {
			return fmt.Errorf("live conformance report case %d has invalid response evidence", index)
		}
		if result.MediaType != "" && !allowedRepresentation(result.MediaType) {
			return fmt.Errorf("live conformance report case %d has invalid media type", index)
		}
		if result.BodyBytes > 0 && result.BodySHA256 == "" || result.BodySHA256 != "" && !validSHA256(result.BodySHA256) {
			return fmt.Errorf("live conformance report case %d has invalid body hash", index)
		}
		if result.Comparison.Kind != "" && (!safeIdentifier.MatchString(result.Comparison.Kind) || result.Comparison.Count < 0) {
			return fmt.Errorf("live conformance report case %d has invalid comparison evidence", index)
		}
		if result.Outcome == "passed" && (result.HTTPStatus != http.StatusOK || result.MediaType != result.Representation || result.BodyBytes == 0 || result.BodySHA256 == "" || result.Comparison.Kind == "" || result.Comparison.Count == 0 || structuredRepresentation(result.Representation) && result.APIStatus != "000") {
			return fmt.Errorf("live conformance report case %d has incomplete success evidence", index)
		}
	}
	if report.Failure != nil {
		expectedOperation, caseExists := caseOperations[report.Failure.CaseID]
		if !allowedFailureCode(report.Failure.Code) || !allowedFailureStage(report.Failure.Code, report.Failure.Stage) || report.Failure.CaseID == "" && report.Failure.Operation != "" || report.Failure.CaseID != "" && (!caseExists || report.Failure.Operation != expectedOperation) {
			return errors.New("live conformance report failure is invalid")
		}
	}
	return nil
}

func allowedFailureStage(code, stage string) bool {
	expected := map[string]string{
		"credential-unavailable":      "credential",
		"request-budget-exhausted":    "budget",
		"request-construction":        "request",
		"transport-failure":           "request",
		"bounded-body-failure":        "response",
		"invalid-media-type":          "response",
		"openapi-response-validation": "response",
		"representation-parse":        "response",
		"unsuccessful-envelope":       "response",
		"assertion-failed":            "assertion",
		"pacing-interrupted":          "pacing",
		"report-sanitization":         "report",
	}
	return expected[code] == stage
}

func allowedRepresentation(value string) bool {
	switch value {
	case "application/json", "application/xml", "application/zip":
		return true
	default:
		return false
	}
}

func allowedFailureCode(value string) bool {
	switch value {
	case "credential-unavailable", "request-budget-exhausted", "request-construction", "transport-failure", "bounded-body-failure", "invalid-media-type", "openapi-response-validation", "representation-parse", "unsuccessful-envelope", "assertion-failed", "pacing-interrupted", "report-sanitization":
		return true
	default:
		return false
	}
}

func validSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
