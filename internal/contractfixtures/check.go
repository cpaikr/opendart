// Package contractfixtures validates the small credential-free protocol corpus.
package contractfixtures

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxBodyBytes = 1 << 20

type corpus struct {
	SchemaVersion int            `json:"schemaVersion"`
	Kind          string         `json:"kind"`
	Observations  []observation  `json:"observations"`
	RequestCases  []requestCase  `json:"requestCases"`
	ResponseCases []responseCase `json:"responseCases"`
}

type observation struct {
	ID         string     `json:"id"`
	Provenance provenance `json:"provenance"`
}

type provenance struct {
	Kind                      string   `json:"kind"`
	ObservedAt                string   `json:"observedAt,omitempty"`
	SourceCommit              string   `json:"sourceCommit,omitempty"`
	RequestCondition          string   `json:"requestCondition,omitempty"`
	HTTPStatus                int      `json:"httpStatus,omitempty"`
	ContentTypeHeader         string   `json:"contentTypeHeader,omitempty"`
	APIStatus                 string   `json:"apiStatus,omitempty"`
	SampledPhysicalOperations []string `json:"sampledPhysicalOperations,omitempty"`
	RetainedOriginalBody      *bool    `json:"retainedOriginalBody,omitempty"`
	Rationale                 string   `json:"rationale,omitempty"`
	Observation               string   `json:"observation,omitempty"`
}

type requestCase struct {
	ID                string `json:"id"`
	PhysicalOperation string `json:"physicalOperation"`
	LogicalOperation  string `json:"logicalOperation"`
	Method            string `json:"method"`
	Path              string `json:"path"`
	Representation    string `json:"representation"`
}

type responseCase struct {
	ID                string     `json:"id"`
	PhysicalOperation string     `json:"physicalOperation"`
	HTTPStatus        int        `json:"httpStatus"`
	Outcome           string     `json:"outcome"`
	File              string     `json:"file"`
	SHA256            string     `json:"sha256"`
	Provenance        provenance `json:"provenance"`
}

// Check validates the canonical fixture manifest and every referenced body.
func Check(repositoryRoot string) error {
	root := filepath.Join(repositoryRoot, "openapi", "fixtures", "v1")
	manifestPath := filepath.Join(root, "manifest.json")
	// #nosec G304 -- the verifier owns this fixed path below the supplied repository root.
	file, err := os.Open(manifestPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	decoder := json.NewDecoder(io.LimitReader(file, maxBodyBytes))
	decoder.DisallowUnknownFields()
	var value corpus
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if err := requireEOF(decoder); err != nil {
		return err
	}
	if value.SchemaVersion != 1 || value.Kind != "opendart-contract-fixtures" {
		return errors.New("unsupported contract fixture manifest")
	}
	pathSources, err := loadPathSources(filepath.Join(repositoryRoot, "openapi", "paths"))
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	observationIDs := make(map[string]struct{})
	observationOperations := make(map[string]map[string]struct{})
	for _, item := range value.Observations {
		if err := uniqueID(seen, item.ID); err != nil {
			return err
		}
		if !completeObservationProvenance(item.Provenance) {
			return fmt.Errorf("observation %s has incomplete provenance", item.ID)
		}
		operations := make(map[string]struct{})
		for _, operation := range item.Provenance.SampledPhysicalOperations {
			source, ok := pathSources[operation]
			if !ok || !observationMatches(source, item.Provenance) {
				return fmt.Errorf("observation %s does not match canonical operation %s", item.ID, operation)
			}
			operations[operation] = struct{}{}
		}
		observationIDs[item.ID] = struct{}{}
		observationOperations[item.ID] = operations
	}
	for _, item := range value.RequestCases {
		if err := uniqueID(seen, item.ID); err != nil {
			return err
		}
		if item.Method != "GET" || !strings.HasPrefix(item.Path, "/api/") || item.PhysicalOperation == "" || item.LogicalOperation == "" {
			return fmt.Errorf("request case %s is incomplete", item.ID)
		}
		if source, ok := pathSources[item.PhysicalOperation]; !ok || !requestMatches(source, item) {
			return fmt.Errorf("request case %s does not match the canonical operation inventory", item.ID)
		}
	}
	referencedBodies := make(map[string]struct{}, len(value.ResponseCases))
	for _, item := range value.ResponseCases {
		if err := uniqueID(seen, item.ID); err != nil {
			return err
		}
		if item.Provenance.Kind != "synthetic" || item.PhysicalOperation == "" || item.HTTPStatus < 100 || item.HTTPStatus > 599 || !responseOutcomeMatchesStatus(item.Outcome, item.HTTPStatus) {
			return fmt.Errorf("response case %s is incomplete", item.ID)
		}
		if item.Provenance.Observation != "" {
			if _, ok := observationIDs[item.Provenance.Observation]; !ok {
				return fmt.Errorf("response case %s references an unknown observation", item.ID)
			}
			if _, ok := observationOperations[item.Provenance.Observation][item.PhysicalOperation]; !ok {
				return fmt.Errorf("response case %s is outside its observation scope", item.ID)
			}
		} else if item.Provenance.Rationale == "" {
			return fmt.Errorf("response case %s has no synthetic rationale", item.ID)
		}
		if _, ok := pathSources[item.PhysicalOperation]; !ok {
			return fmt.Errorf("response case %s references an unknown operation", item.ID)
		}
		if err := verifyBody(root, item); err != nil {
			return err
		}
		referencedBodies[item.File] = struct{}{}
	}
	if err := verifyBodyInventory(root, referencedBodies); err != nil {
		return err
	}
	if err := checkCoverage(value); err != nil {
		return err
	}
	return nil
}

func completeObservationProvenance(value provenance) bool {
	return value.Kind == "empirical-summary" &&
		value.ObservedAt != "" &&
		len(value.SampledPhysicalOperations) > 0 &&
		value.SourceCommit != "" &&
		value.RequestCondition != "" &&
		value.HTTPStatus != 0 &&
		value.ContentTypeHeader != "" &&
		value.APIStatus != ""
}

func checkCoverage(value corpus) error {
	requestRepresentations := make(map[string]bool)
	responseOutcomes := make(map[string]bool)
	typedRepresentations := make(map[string]bool)
	binaryAlternate := false
	for _, item := range value.RequestCases {
		requestRepresentations[item.Representation] = true
	}
	for _, item := range value.ResponseCases {
		responseOutcomes[item.Outcome] = true
		if item.Outcome == "typed-success" {
			switch {
			case strings.HasSuffix(item.PhysicalOperation, "_json"):
				typedRepresentations["json"] = true
			case strings.HasSuffix(item.PhysicalOperation, "_xml"):
				typedRepresentations["xml"] = true
			}
		}
		if item.Outcome == "source-status" && item.Provenance.Observation != "" && item.PhysicalOperation == "get_corpCode_xml" {
			binaryAlternate = true
		}
	}
	if len(value.Observations) == 0 ||
		!requestRepresentations["json"] || !requestRepresentations["zip"] ||
		!typedRepresentations["json"] || !typedRepresentations["xml"] ||
		!responseOutcomes["source-status"] || !responseOutcomes["http-status-failure"] ||
		!responseOutcomes["decode-failure"] || !responseOutcomes["envelope-failure"] ||
		!binaryAlternate {
		return errors.New("contract fixture corpus is missing required protocol-family coverage")
	}
	return nil
}

func requestMatches(source string, item requestCase) bool {
	formats := map[string]string{
		"json": "outputFormat: JSON",
		"xml":  "outputFormat: XML",
		"zip":  "outputFormat: Zip FILE (binary)",
	}
	format, ok := formats[item.Representation]
	return ok &&
		strings.Contains(source, "logicalOperationId: "+item.LogicalOperation) &&
		strings.Contains(source, "method: "+item.Method) &&
		strings.Contains(source, "requestUrl: https://opendart.fss.or.kr"+item.Path) &&
		strings.Contains(source, format)
}

func responseOutcomeMatchesStatus(outcome string, status int) bool {
	success := status >= 200 && status <= 299
	switch outcome {
	case "typed-success", "source-status", "decode-failure", "envelope-failure":
		return success
	case "http-status-failure":
		return !success
	default:
		return false
	}
}

func loadPathSources(root string) (map[string]string, error) {
	result := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("contract fixture inventory encountered a symlink")
		}
		if entry.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}
		// #nosec G304 -- WalkDir roots paths below openapi/paths and rejects symlinks.
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(body) > maxBodyBytes {
			return fmt.Errorf("canonical path file exceeds fixture verification bound")
		}
		source := string(body)
		for _, line := range strings.Split(source, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "operationId: ") {
				result[strings.TrimSpace(strings.TrimPrefix(line, "operationId: "))] = source
			}
		}
		return nil
	})
	return result, err
}

func observationMatches(source string, value provenance) bool {
	wanted := []string{
		"observedAt: " + value.ObservedAt,
		"requestCondition: " + value.RequestCondition,
		fmt.Sprintf("httpStatus: %d", value.HTTPStatus),
		"contentTypeHeader: " + value.ContentTypeHeader,
		"apiStatus: \"" + value.APIStatus + "\"",
	}
	for _, item := range wanted {
		if !strings.Contains(source, item) {
			return false
		}
	}
	return true
}

func verifyBody(root string, item responseCase) error {
	clean := filepath.Clean(filepath.FromSlash(item.File))
	if clean == "." || filepath.IsAbs(clean) || clean != filepath.FromSlash(item.File) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("response case %s has an unsafe body path", item.ID)
	}
	path := filepath.Join(root, clean)
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > maxBodyBytes {
		return fmt.Errorf("response case %s body is not a bounded regular file", item.ID)
	}
	// #nosec G304 -- the normalized relative path was lstat-checked as a bounded regular file.
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.Contains(string(body), "crtfc_key") || strings.Contains(string(body), "OPENDART_API_KEY") {
		return fmt.Errorf("response case %s contains credential material", item.ID)
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != item.SHA256 {
		return fmt.Errorf("response case %s body digest mismatch", item.ID)
	}
	return nil
}

func verifyBodyInventory(root string, referenced map[string]struct{}) error {
	remaining := make(map[string]struct{}, len(referenced))
	for path := range referenced {
		remaining[path] = struct{}{}
	}
	bodiesRoot := filepath.Join(root, "bodies")
	err := filepath.WalkDir(bodiesRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("contract fixture body inventory encountered a symlink")
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return errors.New("contract fixture body inventory contains a non-regular file")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		manifestPath := filepath.ToSlash(relative)
		if _, ok := remaining[manifestPath]; !ok {
			return fmt.Errorf("fixture body %s is absent from the manifest", manifestPath)
		}
		delete(remaining, manifestPath)
		return nil
	})
	if err != nil {
		return err
	}
	for path := range remaining {
		return fmt.Errorf("manifest body %s is outside the body inventory", path)
	}
	return nil
}

func uniqueID(seen map[string]struct{}, id string) error {
	if id == "" {
		return errors.New("contract fixture ID is empty")
	}
	if _, exists := seen[id]; exists {
		return fmt.Errorf("duplicate contract fixture ID %s", id)
	}
	seen[id] = struct{}{}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("contract fixture manifest contains multiple values")
		}
		return err
	}
	return nil
}
