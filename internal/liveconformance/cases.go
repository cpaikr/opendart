package liveconformance

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	samsungCorpCode  = "00126380"
	samsungStockCode = "005930"
)

type logicalCase struct {
	group       string
	stem        string
	assertion   AssertionID
	parameters  map[string][]string
	discovery   DiscoveryID
	detailTypes []string
	aliases     []string
}

// PrimaryCases is the reviewed canonical live matrix. Structured logical rows
// expand into their byte-equivalent JSON and XML physical operations; archive
// operations remain explicit.
func PrimaryCases() []Case {
	definitions := primaryLogicalCases()
	cases := make([]Case, 0, len(definitions)*2+3)
	for _, definition := range definitions {
		for _, representation := range []struct{ suffix, media string }{{"json", "application/json"}, {"xml", "application/xml"}} {
			cases = append(cases, Case{
				ID:             definition.group + "-" + definition.stem + "-" + representation.suffix,
				Method:         http.MethodGet,
				Path:           "/" + definition.stem + "." + representation.suffix,
				Representation: representation.media,
				Parameters:     cloneParameters(definition.parameters),
				Assertion:      definition.assertion,
				Discovery:      definition.discovery,
			})
		}
	}
	cases = append(cases,
		Case{ID: "ds001-corpCode-zip", Method: http.MethodGet, Path: "/corpCode.xml", Representation: "application/zip", Parameters: map[string][]string{}, Assertion: "DS001-2019018"},
		Case{ID: "ds001-document-zip", Method: http.MethodGet, Path: "/document.xml", Representation: "application/zip", Parameters: parameters("rcept_no", "20210414000307"), Assertion: "DS001-2019003"},
		Case{ID: "ds003-fnlttXbrl-zip", Method: http.MethodGet, Path: "/fnlttXbrl.xml", Representation: "application/zip", Parameters: parameters("rcept_no", "20250311001085", "reprt_code", "11011"), Assertion: "DS003-2019019"},
	)
	return cases
}

// PrimaryAssertions returns one named semantic policy per logical operation.
func PrimaryAssertions() map[AssertionID]Assertion {
	assertions := make(map[AssertionID]Assertion)
	for _, definition := range primaryLogicalCases() {
		check := samsungListAssertion
		switch definition.assertion {
		case "DS001-2019001":
			check = samsungListAssertion
		case "DS001-2019002":
			check = companyAssertion
		case "DS003-2020001":
			check = taxonomyAssertion
		default:
			if strings.HasSuffix(definition.stem, "V2") {
				check = samsungRootAssertion
			} else if definition.discovery != "" {
				check = discoveredEventAssertion("list")
				if definition.group == "ds006" {
					check = discoveredEventAssertion("group/list")
				}
			}
		}
		assertions[definition.assertion] = structuredAssertion(check)
	}
	assertions["DS001-2019018"] = Assertion{Representations: []string{"application/zip"}, Check: corporationArchiveAssertion}
	assertions["DS001-2019003"] = Assertion{Representations: []string{"application/zip"}, Check: documentArchiveAssertion}
	assertions["DS003-2019019"] = Assertion{Representations: []string{"application/zip"}, Check: xbrlArchiveAssertion}
	return assertions
}

// PrimaryDiscoveries returns the fixed historical disclosure-list scan used by
// rare event cases. The maximum keeps the complete run below the hard ceiling.
func PrimaryDiscoveries() []Discovery {
	requests := disclosureDiscoveryRequests()
	targets := make([]DiscoveryTarget, 0)
	for _, definition := range primaryLogicalCases() {
		if definition.discovery == "" {
			continue
		}
		for _, suffix := range []string{"json", "xml"} {
			targets = append(targets, DiscoveryTarget{CaseID: definition.group + "-" + definition.stem + "-" + suffix, DetailTypes: append([]string(nil), definition.detailTypes...), Aliases: append([]string(nil), definition.aliases...)})
		}
	}
	return []Discovery{{ID: "rare-disclosures", MaxRequests: len(requests), Requests: requests, Targets: targets}}
}

func structuredAssertion(check func(Response) (ComparisonEvidence, bool)) Assertion {
	return Assertion{Representations: []string{"application/json", "application/xml"}, Check: check}
}

func companyAssertion(response Response) (ComparisonEvidence, bool) {
	count := exactFieldCount(response, "", "stock_code", samsungStockCode)
	return ComparisonEvidence{Kind: "company-stock-identity", Count: count}, count > 0
}

func samsungListAssertion(response Response) (ComparisonEvidence, bool) {
	count := min(exactFieldCount(response, "list", "corp_code", samsungCorpCode), nonemptyFieldCount(response, "list", "rcept_no", regexp.MustCompile(`^[0-9]{14}$`)))
	return ComparisonEvidence{Kind: "samsung-corporation-identity", Count: count}, count > 0
}

func samsungRootAssertion(response Response) (ComparisonEvidence, bool) {
	count := min(exactFieldCount(response, "", "corp_code", samsungCorpCode), nonemptyFieldCount(response, "", "rcept_no", regexp.MustCompile(`^[0-9]{14}$`)))
	return ComparisonEvidence{Kind: "samsung-corporation-identity", Count: count}, count > 0
}

func taxonomyAssertion(response Response) (ComparisonEvidence, bool) {
	count := exactFieldCount(response, "list", "sj_div", "BS1")
	return ComparisonEvidence{Kind: "taxonomy-statement-identity", Count: count}, count > 0
}

func discoveredEventAssertion(container string) func(Response) (ComparisonEvidence, bool) {
	return func(response Response) (ComparisonEvidence, bool) {
		corpCodes := nonemptyFieldCount(response, container, "corp_code", corporationCode)
		receipts := nonemptyFieldCount(response, container, "rcept_no", regexp.MustCompile(`^[0-9]{14}$`))
		count := min(corpCodes, receipts)
		return ComparisonEvidence{Kind: "discovered-event-content", Count: count}, count > 0
	}
}

func eventIdentityAssertion(corpCode, container string) func(Response) (ComparisonEvidence, bool) {
	return func(response Response) (ComparisonEvidence, bool) {
		identities := exactFieldCount(response, container, "corp_code", corpCode)
		receipts := nonemptyFieldCount(response, container, "rcept_no", regexp.MustCompile(`^[0-9]{14}$`))
		count := min(identities, receipts)
		return ComparisonEvidence{Kind: "discovered-event-identity", Count: count}, count > 0
	}
}

func corporationArchiveAssertion(response Response) (ComparisonEvidence, bool) {
	count := archiveValueCount(response, "result/list/corp_code", samsungCorpCode)
	return ComparisonEvidence{Kind: "corporation-archive-identity", Count: count}, count > 0
}

func documentArchiveAssertion(response Response) (ComparisonEvidence, bool) {
	count := archiveAnyValueCount(response, "삼정회계법인")
	return ComparisonEvidence{Kind: "document-auditor-content", Count: count}, count > 0
}

func xbrlArchiveAssertion(response Response) (ComparisonEvidence, bool) {
	count := 0
	for _, document := range response.ArchiveDocuments {
		if strings.EqualFold(document.Root, "xbrl") && len(document.XMLValues["xbrl/context/entity/identifier"]) > 0 {
			count++
		}
	}
	return ComparisonEvidence{Kind: "xbrl-document-content", Count: count}, count > 0
}

func exactFieldCount(response Response, container, field, expected string) int {
	if response.Representation == "application/xml" {
		path := "result/" + field
		if container != "" {
			path = "result/" + container + "/" + field
		}
		count := 0
		for _, value := range response.XMLValues[path] {
			if value == expected {
				count++
			}
		}
		return count
	}
	values := jsonFieldValues(response.JSON, container, field)
	count := 0
	for _, value := range values {
		if value == expected {
			count++
		}
	}
	return count
}

func nonemptyFieldCount(response Response, container, field string, pattern *regexp.Regexp) int {
	if response.Representation == "application/xml" {
		path := "result/" + field
		if container != "" {
			path = "result/" + container + "/" + field
		}
		count := 0
		for _, value := range response.XMLValues[path] {
			if pattern.MatchString(value) {
				count++
			}
		}
		return count
	}
	count := 0
	for _, value := range jsonFieldValues(response.JSON, container, field) {
		if pattern.MatchString(value) {
			count++
		}
	}
	return count
}

func jsonFieldValues(root map[string]any, container, field string) []string {
	if container == "" {
		value, _ := root[field].(string)
		return []string{value}
	}
	nodes := []any{root}
	for _, component := range strings.Split(container, "/") {
		next := make([]any, 0)
		for _, node := range nodes {
			switch current := node.(type) {
			case map[string]any:
				if value, exists := current[component]; exists {
					next = append(next, value)
				}
			case []any:
				for _, item := range current {
					object, _ := item.(map[string]any)
					if value, exists := object[component]; exists {
						next = append(next, value)
					}
				}
			}
		}
		nodes = next
	}
	values := make([]string, 0)
	for _, node := range nodes {
		items, ok := node.([]any)
		if !ok {
			items = []any{node}
		}
		for _, item := range items {
			row, _ := item.(map[string]any)
			value, _ := row[field].(string)
			values = append(values, value)
		}
	}
	return values
}

func archiveValueCount(response Response, path, expected string) int {
	count := 0
	for _, document := range response.ArchiveDocuments {
		for _, value := range document.XMLValues[path] {
			if value == expected {
				count++
			}
		}
	}
	return count
}

func archiveAnyValueCount(response Response, expected string) int {
	count := 0
	for _, document := range response.ArchiveDocuments {
		for _, values := range document.XMLValues {
			for _, value := range values {
				count += strings.Count(value, expected)
			}
		}
	}
	return count
}

func parameters(values ...string) map[string][]string {
	result := make(map[string][]string, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		result[values[index]] = []string{values[index+1]}
	}
	return result
}

func cloneParameters(source map[string][]string) map[string][]string {
	return cloneCase(Case{Parameters: source}).Parameters
}
func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
