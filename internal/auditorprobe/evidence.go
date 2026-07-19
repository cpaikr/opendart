package auditorprobe

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"
)

const maximumEvidenceFile = 8 << 20

// ValidateEvidenceFile verifies that a committed manifest is a complete,
// sanitized result of the fixed probe matrix. It performs no network access.
func ValidateEvidenceFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open auditor evidence: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat auditor evidence: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maximumEvidenceFile {
		return errors.New("auditor evidence is not a bounded regular file")
	}

	limited := io.LimitReader(file, maximumEvidenceFile)
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	var report Report
	if err := decoder.Decode(&report); err != nil {
		return fmt.Errorf("decode auditor evidence: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("decode auditor evidence: expected one JSON value")
	}
	return validateEvidence(report)
}

func validateEvidence(report Report) error {
	if report.SchemaVersion != reportSchemaVersion {
		return errors.New("auditor evidence has an unsupported schema version")
	}
	observedAt, err := time.Parse("2006-01-02T15:04:05.000Z", report.ObservedAt)
	if err != nil {
		return errors.New("auditor evidence has an invalid observation timestamp")
	}
	observedDate, err := checkedAtInSeoul(observedAt)
	if err != nil || observedDate != report.ObservedDate {
		return errors.New("auditor evidence observation date does not match its timestamp")
	}
	if report.ObservedDate != "2026-07-18" {
		return errors.New("auditor evidence observation date does not match its dated artifact")
	}
	if report.CredentialSource != apiKeyEnvironment || report.CredentialPersisted {
		return errors.New("auditor evidence violates the credential boundary")
	}
	if report.RequestBudget.Maximum != maximumRequestBudget || report.RequestBudget.Used <= 0 || report.RequestBudget.Used > maximumRequestBudget {
		return errors.New("auditor evidence has an invalid request budget")
	}
	if len(report.Structured) != len(structuredCases) || len(report.Searches) != len(searchCases) || len(report.Documents) != 4 {
		return errors.New("auditor evidence does not cover the fixed probe matrix")
	}
	used := len(report.Structured) + len(report.Documents)
	for index, observation := range report.Structured {
		probeCase := structuredCases[index]
		want := RequestCoordinate{
			LogicalOperationID: probeCase.LogicalID,
			Endpoint:           probeCase.Endpoint + ".json",
			CompanyName:        probeCase.Company.Name,
			CorpCode:           probeCase.Company.CorpCode,
			BusinessYear:       probeCase.BusinessYear,
			ReportCode:         probeCase.ReportCode,
		}
		if observation.Request != want {
			return fmt.Errorf("auditor evidence structured coordinate %d is not canonical", index)
		}
		if err := validateResponseEvidence(observation.Response, maxJSONBody); err != nil {
			return fmt.Errorf("auditor evidence structured response %d: %w", index, err)
		}
		status := stringValue(observation.Response.APIStatus)
		if observation.Response.HTTPStatus != 200 || stringValue(observation.Response.MediaType) != "application/json" {
			return fmt.Errorf("auditor evidence structured response %d has an unexpected media type", index)
		}
		if status != "000" && status != "013" {
			return fmt.Errorf("auditor evidence structured response %d has an unexpected API status", index)
		}
		if observation.RowCount < 0 || observation.PlaceholderCount < 0 || observation.PlaceholderCount > 2*observation.RowCount ||
			len(observation.ReceiptNumbers) > observation.RowCount || len(observation.DistinctAuditors) > observation.RowCount || len(observation.DistinctOpinions) > observation.RowCount ||
			status == "013" && (observation.RowCount != 0 || observation.PlaceholderCount != 0 || len(observation.ReceiptNumbers) != 0 || len(observation.DistinctAuditors) != 0 || len(observation.DistinctOpinions) != 0) {
			return fmt.Errorf("auditor evidence structured response %d has inconsistent row counts", index)
		}
		if !validSortedReceiptSet(observation.ReceiptNumbers) || !validSortedSubstantiveSet(observation.DistinctAuditors) || !validSortedSubstantiveSet(observation.DistinctOpinions) {
			return fmt.Errorf("auditor evidence structured response %d has invalid distinct values", index)
		}
	}
	for index, observation := range report.Searches {
		probeCase := searchCases[index]
		if observation.CompanyName != probeCase.Company.Name || observation.CorpCode != probeCase.Company.CorpCode || observation.DetailType != probeCase.DetailType {
			return fmt.Errorf("auditor evidence search %d is not canonical", index)
		}
		if len(observation.Pages) == 0 || len(observation.Pages) > maxSearchPages {
			return fmt.Errorf("auditor evidence search %d has invalid pagination", index)
		}
		used += len(observation.Pages)
		expectedTotal, observedRows := -1, 0
		seenReceipts := make(map[string]bool)
		for pageIndex, page := range observation.Pages {
			wantPage := pageIndex + 1
			want := RequestCoordinate{
				LogicalOperationID: "DS001-2019001", Endpoint: "list.json",
				CompanyName: probeCase.Company.Name, CorpCode: probeCase.Company.CorpCode,
				BeginDate: searchBeginDate, EndDate: searchEndDate, LastReport: "N",
				PublicationType: "F", DetailType: probeCase.DetailType,
				PageNumber: wantPage, PageCount: searchPageCount,
			}
			if page.Request != want || page.PageNumber != wantPage || page.PageCount != searchPageCount {
				return fmt.Errorf("auditor evidence search %d page %d is not canonical", index, wantPage)
			}
			if err := validateResponseEvidence(page.Response, maxJSONBody); err != nil {
				return fmt.Errorf("auditor evidence search %d page %d: %w", index, wantPage, err)
			}
			status := stringValue(page.Response.APIStatus)
			if page.Response.HTTPStatus != 200 || stringValue(page.Response.MediaType) != "application/json" {
				return fmt.Errorf("auditor evidence search %d page %d has an unexpected media type", index, wantPage)
			}
			switch status {
			case "013":
				if len(observation.Pages) != 1 || len(page.Filings) != 0 || page.TotalCount != 0 || page.TotalPages != 0 {
					return fmt.Errorf("auditor evidence search %d has inconsistent empty pagination", index)
				}
			case "000":
				if page.TotalPages != len(observation.Pages) || page.TotalCount < len(page.Filings) || page.TotalPages < 1 {
					return fmt.Errorf("auditor evidence search %d has incomplete pagination", index)
				}
				if expectedTotal == -1 {
					expectedTotal = page.TotalCount
				} else if expectedTotal != page.TotalCount {
					return fmt.Errorf("auditor evidence search %d has inconsistent total counts", index)
				}
				observedRows += len(page.Filings)
				for _, filing := range page.Filings {
					if filing.CorpCode != observation.CorpCode || !validReceiptNumber(filing.ReceiptNumber) || seenReceipts[filing.ReceiptNumber] {
						return fmt.Errorf("auditor evidence search %d contains an invalid filing", index)
					}
					seenReceipts[filing.ReceiptNumber] = true
				}
			default:
				return fmt.Errorf("auditor evidence search %d has an unexpected API status", index)
			}
		}
		if expectedTotal >= 0 && observedRows != expectedTotal {
			return fmt.Errorf("auditor evidence search %d does not account for every reported row", index)
		}
	}
	wantDocuments := []struct {
		company company
		period  string
	}{
		{nuga, "2015"}, {nuga, "2020"}, {nuga, "2025"}, {lotte, "2024"},
	}
	canonicalSelections, err := documentSelections(report.Structured, report.Searches)
	if err != nil || len(canonicalSelections) != len(report.Documents) {
		return errors.New("auditor evidence document selections could not be reproduced")
	}
	for index, document := range report.Documents {
		want := wantDocuments[index]
		if document.Selection != canonicalSelections[index] || document.Selection.CompanyName != want.company.Name || document.Selection.ReportPeriod != want.period || !validReceiptNumber(document.Selection.ReceiptNumber) || document.Selection.ExpectedFirm == "" {
			return fmt.Errorf("auditor evidence document %d is not canonical", index)
		}
		coordinate := RequestCoordinate{
			LogicalOperationID: "DS001-2019003", Endpoint: "document.xml",
			CompanyName: want.company.Name, ReceiptNumber: document.Selection.ReceiptNumber,
		}
		if document.Request != coordinate || !supportedDocumentMediaType(document.Response.MediaType) {
			return fmt.Errorf("auditor evidence document %d has invalid response identity", index)
		}
		if err := validateResponseEvidence(document.Response, maxArchiveBody); err != nil {
			return fmt.Errorf("auditor evidence document %d: %w", index, err)
		}
		if document.Response.HTTPStatus != 200 || document.Response.APIStatus != nil {
			return fmt.Errorf("auditor evidence document %d has unexpected response status", index)
		}
		if document.ArchiveBytes != document.Response.BodyBytes || document.ArchiveSHA256 != document.Response.BodySHA256 ||
			document.EntryCount != len(document.Entries) || document.EntryCount == 0 || document.EntryCount > maxArchiveEntries ||
			document.ExpandedBytes <= 0 || document.ExpandedBytes > maxArchiveExpanded || document.ExpectedMatches <= 0 || document.SectionMarkers <= 0 ||
			!hasCoLocatedAuditorEvidence(document.Entries) {
			return fmt.Errorf("auditor evidence document %d lacks a verified auditor section", index)
		}
		var expanded int64
		expectedMatches, sectionMarkers := 0, 0
		for _, entry := range document.Entries {
			if entry.Name == "" || entry.ExpandedBytes > maxArchiveMember || entry.CompressedBytes > uint64(document.ArchiveBytes) ||
				entry.ExpectedFirmMatches < 0 || entry.AuditorSectionMarkerMatches < 0 ||
				uint64(entry.ExpectedFirmMatches) > entry.ExpandedBytes || uint64(entry.AuditorSectionMarkerMatches) > entry.ExpandedBytes ||
				!validSHA256(entry.NameSHA256) || !validSHA256(entry.ContentSHA256) || !slices.Contains([]string{"utf-8", "cp949-compatible", "binary-or-unsupported"}, entry.Decoding) {
				return fmt.Errorf("auditor evidence document %d has invalid entry evidence", index)
			}
			expanded += int64(entry.ExpandedBytes)
			expectedMatches += entry.ExpectedFirmMatches
			sectionMarkers += entry.AuditorSectionMarkerMatches
		}
		if expanded != document.ExpandedBytes || expectedMatches != document.ExpectedMatches || sectionMarkers != document.SectionMarkers {
			return fmt.Errorf("auditor evidence document %d has inconsistent entry aggregates", index)
		}
	}
	if used != report.RequestBudget.Used {
		return errors.New("auditor evidence request count does not match the manifest")
	}
	contains, err := reportContainsForbiddenRequestMaterial(report)
	if err != nil {
		return errors.New("auditor evidence could not be inspected")
	}
	if contains {
		return errors.New("auditor evidence contains forbidden request material")
	}
	return nil
}

func validSortedReceiptSet(values []string) bool {
	for index, value := range values {
		if !validReceiptNumber(value) || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func validSortedSubstantiveSet(values []string) bool {
	for index, value := range values {
		if isPlaceholder(value) || strings.TrimSpace(value) != value || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func validateResponseEvidence(evidence ResponseEvidence, maximumBody int) error {
	if evidence.HTTPStatus < 100 || evidence.HTTPStatus > 599 || evidence.BodyBytes <= 0 || evidence.BodyBytes > maximumBody || !validSHA256(evidence.BodySHA256) {
		return errors.New("invalid bounded response evidence")
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}
