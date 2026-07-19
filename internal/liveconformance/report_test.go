package liveconformance

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDecodeReportAcceptsOnlyStrictSanitizedSchema(t *testing.T) {
	report := newReport(time.Date(2026, 7, 19, 1, 2, 3, 0, time.UTC), 1)
	report.Outcome = "passed"
	report.RequestBudget.Used = 1
	report.Cases = []CaseResult{{
		CaseID: "company-json", OperationID: "get_company_json", LogicalOperationID: "DS001-2019002",
		Method: "GET", Path: "/company.json", Representation: "application/json", AssertionID: "company-identity",
		Outcome: "passed", HTTPStatus: 200, MediaType: "application/json", APIStatus: "000", BodyBytes: 12,
		BodySHA256: strings.Repeat("a", 64), SchemaLocation: "/company.json#responses/default/content/application~1json",
		Comparison: ComparisonEvidence{Kind: "company-identity", Count: 1},
	}}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeReport(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Outcome != "passed" || len(decoded.Cases) != 1 {
		t.Fatalf("decoded report = %#v", decoded)
	}

	for _, mutation := range []struct {
		name    string
		content []byte
	}{
		{name: "unknown raw body", content: bytes.Replace(encoded, []byte(`"bodyBytes":12`), []byte(`"rawBody":"secret","bodyBytes":12`), 1)},
		{name: "authenticated path", content: bytes.Replace(encoded, []byte(`"path":"/company.json"`), []byte(`"path":"/company.json?crtfc_key=secret"`), 1)},
		{name: "trailing document", content: append(append([]byte(nil), encoded...), []byte(`{}`)...)},
		{name: "oversized", content: bytes.Repeat([]byte("x"), MaximumReportBytes+1)},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			if _, err := DecodeReport(bytes.NewReader(mutation.content)); err == nil {
				t.Fatal("DecodeReport() accepted unsafe content")
			}
		})
	}
}
