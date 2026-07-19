package liveconformance

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

var (
	corporationCode = regexp.MustCompile(`^[0-9]{8}$`)
	receiptDate     = regexp.MustCompile(`^[0-9]{8}$`)
)

type discoveredCoordinate struct {
	corpCode  string
	beginDate string
	endDate   string
}

func executeDiscoveries(ctx context.Context, plan *Plan, deps dependencies, credential string, report *Report) ([]preparedCase, *Failure) {
	resolved := append([]preparedCase(nil), plan.cases...)
	caseIndex := make(map[string]int, len(resolved))
	for index := range resolved {
		resolved[index].query = cloneValues(resolved[index].query)
		caseIndex[resolved[index].definition.ID] = index
	}
	for _, discovery := range plan.discoveries {
		coordinates := make(map[string]discoveredCoordinate)
		pageCeilings := declaredPageCeilings(discovery.requests)
		observedPages := make(map[string]int)
		for _, request := range discovery.requests {
			partition := discoveryPartitionKey(request.query)
			page, _ := strconv.Atoi(request.query.Get("page_no"))
			if totalPages, known := observedPages[partition]; known && page > totalPages {
				continue
			}
			if report.RequestBudget.Used >= report.RequestBudget.Ceiling {
				return nil, discoveryFailure("discovery-budget-exhausted", discovery.definition.ID)
			}
			_, response, failure := executeCase(ctx, plan.specification, deps.do, credential, request)
			report.RequestBudget.Used++
			report.RequestBudget.DiscoveryUsed++
			if failure != nil {
				return nil, discoveryFailure("discovery-request-failed", discovery.definition.ID)
			}
			if !discoveryPagesClosed(response, request.query, pageCeilings, observedPages) {
				return nil, discoveryFailure("discovery-pagination-open", discovery.definition.ID)
			}
			if response.APIStatus == "013" {
				observedPages[partition] = 0
			} else if totalPages, ok := jsonInteger(response.JSON["total_page"]); ok {
				observedPages[partition] = totalPages
			}
			collectDiscoveryCoordinates(response, request.query, discovery.definition.Targets, coordinates)
			if err := deps.wait(RequestPacing); err != nil {
				return nil, discoveryFailure("discovery-pacing-interrupted", discovery.definition.ID)
			}
		}
		for _, target := range discovery.definition.Targets {
			coordinate, exists := coordinates[target.CaseID]
			index, caseExists := caseIndex[target.CaseID]
			if !exists || !caseExists {
				return nil, discoveryFailure("discovery-incomplete", discovery.definition.ID)
			}
			query := cloneValues(resolved[index].query)
			query.Set("corp_code", coordinate.corpCode)
			query.Set("bgn_de", coordinate.beginDate)
			query.Set("end_de", coordinate.endDate)
			if err := plan.specification.ValidateRequest(http.MethodGet, resolved[index].operation.Path, query); err != nil {
				return nil, discoveryFailure("discovery-coordinate-invalid", discovery.definition.ID)
			}
			resolved[index].query = query
			container := "list"
			if strings.HasPrefix(resolved[index].operation.LogicalOperationID, "DS006-") {
				container = "group/list"
			}
			resolved[index].assertion = structuredAssertion(eventIdentityAssertion(coordinate.corpCode, container))
		}
	}
	return resolved, nil
}

func declaredPageCeilings(requests []preparedCase) map[string]int {
	ceilings := make(map[string]int)
	for _, request := range requests {
		page, err := strconv.Atoi(request.query.Get("page_no"))
		if err != nil {
			continue
		}
		key := discoveryPartitionKey(request.query)
		if page > ceilings[key] {
			ceilings[key] = page
		}
	}
	return ceilings
}

func discoveryPagesClosed(response Response, query url.Values, ceilings, observed map[string]int) bool {
	wantPage, err := strconv.Atoi(query.Get("page_no"))
	if err != nil {
		return false
	}
	partition := discoveryPartitionKey(query)
	if response.APIStatus == "013" {
		_, alreadyObserved := observed[partition]
		return wantPage == 1 && !alreadyObserved
	}
	page, pageOK := jsonInteger(response.JSON["page_no"])
	totalPages, totalOK := jsonInteger(response.JSON["total_page"])
	if !pageOK || !totalOK || page != wantPage || totalPages < wantPage || totalPages > ceilings[partition] {
		return false
	}
	previous, alreadyObserved := observed[partition]
	return !alreadyObserved || previous == totalPages
}

func discoveryPartitionKey(query url.Values) string {
	return strings.Join([]string{query.Get("pblntf_detail_ty"), query.Get("bgn_de"), query.Get("end_de")}, "|")
}

func jsonInteger(value any) (int, bool) {
	switch current := value.(type) {
	case string:
		parsed, err := strconv.Atoi(current)
		return parsed, err == nil && parsed >= 0
	case float64:
		parsed := int(current)
		return parsed, current == float64(parsed) && parsed >= 0
	default:
		return 0, false
	}
}

func collectDiscoveryCoordinates(response Response, query url.Values, targets []DiscoveryTarget, coordinates map[string]discoveredCoordinate) {
	rows, _ := response.JSON["list"].([]any)
	for _, item := range rows {
		row, _ := item.(map[string]any)
		corpCode, _ := row["corp_code"].(string)
		reportName, _ := row["report_nm"].(string)
		received, _ := row["rcept_dt"].(string)
		if !corporationCode.MatchString(corpCode) || !receiptDate.MatchString(received) || correctedReportName(reportName) {
			continue
		}
		name := normalizeReportName(reportName)
		detailType := query.Get("pblntf_detail_ty")
		for _, target := range targets {
			if _, exists := coordinates[target.CaseID]; exists || !slices.Contains(target.DetailTypes, detailType) {
				continue
			}
			for _, alias := range target.Aliases {
				if name == normalizeReportName(alias) {
					coordinates[target.CaseID] = discoveredCoordinate{corpCode: corpCode, beginDate: firstQueryValue(query, "bgn_de"), endDate: firstQueryValue(query, "end_de")}
					break
				}
			}
		}
	}
}

func correctedReportName(value string) bool {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{"[기재정정]", "[첨부정정]", "[첨부추가]", "[발행조건확정]"} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func normalizeReportName(value string) string {
	value = strings.TrimSpace(value)
	for {
		original := value
		for _, prefix := range []string{"[기재정정]", "[첨부정정]", "[첨부추가]", "[발행조건확정]"} {
			value = strings.TrimPrefix(value, prefix)
		}
		if value == original {
			break
		}
	}
	var normalized strings.Builder
	for _, current := range value {
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			normalized.WriteRune(current)
		}
	}
	result := normalized.String()
	for _, prefix := range []string{"주요사항보고서", "증권신고서", "소액공모공시서류", "증권발행실적보고서"} {
		if strings.HasPrefix(result, prefix) && len(result) > len(prefix) {
			result = strings.TrimPrefix(result, prefix)
		}
	}
	return result
}

func discoveryEnvelopeAssertion(response Response) (ComparisonEvidence, bool) {
	if response.APIStatus == "013" {
		return ComparisonEvidence{Kind: "discovery-empty-envelope", Count: 1}, true
	}
	rows, ok := response.JSON["list"].([]any)
	return ComparisonEvidence{Kind: "discovery-list-content", Count: len(rows)}, ok && len(rows) > 0
}

func firstQueryValue(query url.Values, name string) string {
	values := query[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func discoveryFailure(code string, id DiscoveryID) *Failure {
	return &Failure{Code: code, Stage: "discovery", DiscoveryID: id}
}
