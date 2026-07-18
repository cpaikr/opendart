// Package multicompanyprobe owns the focused live check for OpenDART's two
// documented multi-company request encodings. It is intentionally narrower
// than the planned general live-conformance runner.
package multicompanyprobe

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

const (
	apiOrigin           = "https://opendart.fss.or.kr/api"
	apiKeyEnvironment   = "OPENDART_API_KEY"
	requestTimeout      = 30 * time.Second
	requestPacing       = 100 * time.Millisecond
	maxResponseBody     = 8 << 20
	probeUserAgent      = "opendart-serialization-probe/1.0"
	reportSchemaVersion = 1
)

type probeCase struct {
	LogicalOperationID    string
	Endpoint              string
	CorpCodes             []string
	ResponseIdentityField string
	Arguments             map[string]string
}

var cases = []probeCase{
	{
		LogicalOperationID:    "DS003-2019017",
		Endpoint:              "fnlttMultiAcnt",
		CorpCodes:             []string{"00334624", "00126380"},
		ResponseIdentityField: "stock_code",
		Arguments: map[string]string{
			"bsns_year":  "2018",
			"reprt_code": "11011",
		},
	},
	{
		LogicalOperationID:    "DS003-2022002",
		Endpoint:              "fnlttCmpnyIndx",
		CorpCodes:             []string{"00164742", "00159023"},
		ResponseIdentityField: "corp_code",
		Arguments: map[string]string{
			"bsns_year":   "2023",
			"reprt_code":  "11014",
			"idx_cl_code": "M210000",
		},
	},
}

type serialization string

const (
	commaSeparated      serialization = "comma-separated"
	repeatedQueryKey    serialization = "repeated-query-key"
	singleValueBaseline serialization = "single-value-baseline"
)

// Report is the versioned, allowlisted observation emitted by the focused
// probe. It deliberately contains neither authenticated URLs nor response
// bodies.
type Report struct {
	SchemaVersion       int           `json:"schemaVersion"`
	ObservedAt          string        `json:"observedAt"`
	ObservedDate        string        `json:"observedDate"`
	CredentialSource    string        `json:"credentialSource"`
	CredentialPersisted bool          `json:"credentialPersisted"`
	RequestCount        int           `json:"requestCount"`
	Baselines           []Observation `json:"baselines"`
	Canonical           []Observation `json:"canonical"`
	RepeatedKeyControls []Observation `json:"repeatedKeyControls"`
	Conclusion          Conclusion    `json:"conclusion"`
}

type Conclusion struct {
	CommaSeparated           string `json:"commaSeparated"`
	RepeatedQueryKey         string `json:"repeatedQueryKey"`
	RepeatedQueryKeyContract string `json:"repeatedQueryKeyContract"`
}

type Observation struct {
	Request  RequestObservation  `json:"request"`
	Response ResponseObservation `json:"response"`
}

type RequestObservation struct {
	LogicalOperationID    string            `json:"logicalOperationId"`
	Endpoint              string            `json:"endpoint"`
	Format                string            `json:"format"`
	Serialization         string            `json:"serialization"`
	ResponseIdentityField string            `json:"responseIdentityField"`
	CorpCodes             []string          `json:"corpCodes"`
	Arguments             map[string]string `json:"arguments"`
}

type ResponseObservation struct {
	HTTPStatus                    int      `json:"httpStatus"`
	MediaType                     *string  `json:"mediaType"`
	APIStatus                     *string  `json:"apiStatus"`
	ReturnedIdentityValues        []string `json:"returnedIdentityValues"`
	MissingExpectedIdentityValues []string `json:"missingExpectedIdentityValues"`
	UnexpectedIdentityValues      []string `json:"unexpectedIdentityValues"`
}

// Error is a sanitized probe failure. Context is restricted to request
// identity and allowlisted response metadata.
type Error struct {
	Message string         `json:"message"`
	Context map[string]any `json:"context"`
}

func (e *Error) Error() string { return e.Message }

type responseValidator interface {
	ValidateResponse(method, path, contentType string, status int, body []byte) error
}

type dependencies struct {
	client   *http.Client
	validate responseValidator
	now      func() time.Time
	wait     func(context.Context, time.Duration) error
	key      string
}

// Run executes the focused probe using the canonical repository OpenAPI
// document and the credential from OPENDART_API_KEY.
func Run(ctx context.Context, repositoryRoot string) (Report, error) {
	key, err := apiKey()
	if err != nil {
		return Report{}, err
	}
	document, err := openapispec.Load(filepath.Join(repositoryRoot, "openapi", "openapi.yaml"))
	if err != nil {
		return Report{}, newError("OpenAPI response validator could not be initialized", nil)
	}
	defer document.Close()

	return run(ctx, dependencies{
		client:   newHTTPClient(),
		validate: document,
		now:      time.Now,
		wait:     waitFor,
		key:      key,
	})
}

func newHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// net/http may transparently replay idempotent requests when a reused
	// connection fails. A fresh HTTP/1 connection per observation makes the
	// probe's one-attempt policy a transport invariant as well as a runner rule.
	transport.DisableKeepAlives = true
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("redirects are not allowed")
		},
	}
}

func run(ctx context.Context, deps dependencies) (Report, error) {
	baselines := make([]Observation, 0, 2)
	canonical := make([]Observation, 0, 4)
	controls := make([]Observation, 0, 4)

	for _, testCase := range cases {
		expected := append([]string(nil), testCase.CorpCodes...)
		if testCase.ResponseIdentityField != "corp_code" {
			caseBaselines := make([]Observation, 0, len(testCase.CorpCodes))
			for _, corpCode := range testCase.CorpCodes {
				baselineCase := testCase
				baselineCase.CorpCodes = []string{corpCode}
				observation, err := observe(ctx, deps, baselineCase, "json", singleValueBaseline, nil)
				if err != nil {
					return Report{}, err
				}
				caseBaselines = append(caseBaselines, observation)
				if err := deps.wait(ctx, requestPacing); err != nil {
					return Report{}, newError("Probe pacing was interrupted", requestContext(testCase, "json", singleValueBaseline))
				}
			}
			var err error
			expected, err = distinctBaselineIdentityValues(testCase, caseBaselines)
			if err != nil {
				return Report{}, err
			}
			baselines = append(baselines, caseBaselines...)
		}

		for _, format := range []string{"json", "xml"} {
			comma, err := observe(ctx, deps, testCase, format, commaSeparated, expected)
			if err != nil {
				return Report{}, err
			}
			if err := assertCanonicalObservation(comma); err != nil {
				return Report{}, err
			}
			canonical = append(canonical, comma)
			if err := deps.wait(ctx, requestPacing); err != nil {
				return Report{}, newError("Probe pacing was interrupted", requestContext(testCase, format, commaSeparated))
			}

			control, err := observe(ctx, deps, testCase, format, repeatedQueryKey, expected)
			if err != nil {
				return Report{}, err
			}
			controls = append(controls, control)
			if err := deps.wait(ctx, requestPacing); err != nil {
				return Report{}, newError("Probe pacing was interrupted", requestContext(testCase, format, repeatedQueryKey))
			}
		}
	}

	observedAt := deps.now().UTC().Truncate(time.Millisecond)
	observedDate, err := checkedAtInSeoul(observedAt)
	if err != nil {
		return Report{}, newError("Probe observation date could not be determined", nil)
	}
	report := Report{
		SchemaVersion:       reportSchemaVersion,
		ObservedAt:          observedAt.Format("2006-01-02T15:04:05.000Z"),
		ObservedDate:        observedDate,
		CredentialSource:    apiKeyEnvironment,
		CredentialPersisted: false,
		RequestCount:        len(baselines) + len(canonical) + len(controls),
		Baselines:           baselines,
		Canonical:           canonical,
		RepeatedKeyControls: controls,
		Conclusion: Conclusion{
			CommaSeparated:           "verified",
			RepeatedQueryKey:         repeatedKeyConclusion(controls),
			RepeatedQueryKeyContract: "non-canonical-control-only",
		},
	}
	containsCredential, err := reportContainsCredential(report, deps.key)
	if err != nil {
		return Report{}, newError("Sanitized probe report could not be inspected", nil)
	}
	if containsCredential {
		return Report{}, newError("Sanitized probe report unexpectedly contains the API key", nil)
	}
	return report, nil
}

func observe(ctx context.Context, deps dependencies, testCase probeCase, format string, encoding serialization, expected []string) (Observation, error) {
	contextFields := requestContext(testCase, format, encoding)
	requestContextWithTimeout, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContextWithTimeout, http.MethodGet, encodedQuery(testCase, format, encoding, deps.key), nil)
	if err != nil {
		return Observation{}, newError("OpenDART request could not be constructed", contextFields)
	}
	request.Header.Set("Accept", map[string]string{"json": "application/json", "xml": "application/xml"}[format])
	request.Header.Set("User-Agent", probeUserAgent)
	response, err := deps.client.Do(request)
	if err != nil {
		return Observation{}, newError("OpenDART request failed before a response was received", contextFields)
	}
	body, err := readBoundedBody(response.Body)
	if err != nil {
		return Observation{}, newError(err.Error(), contextFields)
	}
	contentType := response.Header.Get("Content-Type")
	if err := deps.validate.ValidateResponse(http.MethodGet, "/"+testCase.Endpoint+"."+format, contentType, response.StatusCode, body); err != nil {
		return Observation{}, newError("OpenDART response did not satisfy the committed OpenAPI representation", contextFields)
	}

	parsed, err := parseObservation(body, format, testCase.ResponseIdentityField, contextFields)
	if err != nil {
		return Observation{}, err
	}
	missing, unexpected := identityDifferences(expected, parsed.returnedIdentityValues)
	media := mediaType(contentType)
	return Observation{
		Request: RequestObservation{
			LogicalOperationID:    testCase.LogicalOperationID,
			Endpoint:              testCase.Endpoint,
			Format:                format,
			Serialization:         string(encoding),
			ResponseIdentityField: testCase.ResponseIdentityField,
			CorpCodes:             append([]string(nil), testCase.CorpCodes...),
			Arguments:             cloneMap(testCase.Arguments),
		},
		Response: ResponseObservation{
			HTTPStatus:                    response.StatusCode,
			MediaType:                     media,
			APIStatus:                     parsed.apiStatus,
			ReturnedIdentityValues:        parsed.returnedIdentityValues,
			MissingExpectedIdentityValues: missing,
			UnexpectedIdentityValues:      unexpected,
		},
	}, nil
}

func apiKey() (string, error) {
	key := os.Getenv(apiKeyEnvironment)
	if len(key) != 40 {
		return "", newError(apiKeyEnvironment+" must contain the 40-character OpenDART API key", nil)
	}
	return key, nil
}

func encodedQuery(testCase probeCase, format string, encoding serialization, key string) string {
	pairs := []string{"crtfc_key=" + queryEscape(key)}
	if encoding == commaSeparated {
		pairs = append(pairs, "corp_code="+strings.Join(testCase.CorpCodes, ","))
	} else {
		for _, corpCode := range testCase.CorpCodes {
			pairs = append(pairs, "corp_code="+queryEscape(corpCode))
		}
	}
	for _, name := range []string{"bsns_year", "reprt_code", "idx_cl_code"} {
		if value, ok := testCase.Arguments[name]; ok {
			pairs = append(pairs, name+"="+queryEscape(value))
		}
	}
	return apiOrigin + "/" + testCase.Endpoint + "." + format + "?" + strings.Join(pairs, "&")
}

func queryEscape(value string) string {
	escaped := url.QueryEscape(value)
	return strings.NewReplacer(
		"+", "%20",
		"%21", "!",
		"%27", "'",
		"%28", "(",
		"%29", ")",
		"%2A", "*",
	).Replace(escaped)
}

type parsedObservation struct {
	apiStatus              *string
	returnedIdentityValues []string
}

func parseObservation(body []byte, format, identityField string, contextFields map[string]any) (parsedObservation, error) {
	if format == "json" {
		return jsonObservation(body, identityField, contextFields)
	}
	return xmlObservation(body, identityField, contextFields)
}

func jsonObservation(body []byte, identityField string, contextFields map[string]any) (parsedObservation, error) {
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		return parsedObservation{}, newError("OpenDART returned invalid JSON", contextFields)
	}
	var status *string
	if text, ok := value["status"].(string); ok {
		status = &text
	}
	identities := make(map[string]bool)
	list, _ := value["list"].([]any)
	for _, item := range list {
		row, _ := item.(map[string]any)
		if identity, ok := row[identityField].(string); ok && identity != "" {
			identities[identity] = true
		}
	}
	return parsedObservation{apiStatus: status, returnedIdentityValues: sortedKeys(identities)}, nil
}

func xmlObservation(body []byte, identityField string, contextFields map[string]any) (parsedObservation, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	depth := 0
	rootSeen := false
	rootClosed := false
	listDepths := make([]int, 0)
	statusDepth := 0
	identityDepth := 0
	statusSeen := false
	var statusText strings.Builder
	var identityText strings.Builder
	var status *string
	identities := make(map[string]bool)
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return parsedObservation{}, newError("OpenDART returned malformed XML", contextFields)
		}
		switch current := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				if rootSeen || rootClosed {
					return parsedObservation{}, newError("OpenDART returned malformed XML", contextFields)
				}
				rootSeen = true
			}
			depth++
			if current.Name.Local == "list" {
				listDepths = append(listDepths, depth)
			}
			if current.Name.Local == "status" && !statusSeen {
				statusSeen = true
				statusDepth = depth
				statusText.Reset()
			}
			if len(listDepths) > 0 && current.Name.Local == identityField {
				identityDepth = depth
				identityText.Reset()
			}
		case xml.CharData:
			if statusDepth > 0 {
				statusText.Write([]byte(current))
			}
			if identityDepth > 0 {
				identityText.Write([]byte(current))
			}
		case xml.EndElement:
			if statusDepth == depth && current.Name.Local == "status" {
				if trimmed := strings.TrimSpace(statusText.String()); trimmed != "" {
					status = &trimmed
				}
				statusDepth = 0
			}
			if identityDepth == depth && current.Name.Local == identityField {
				if trimmed := strings.TrimSpace(identityText.String()); trimmed != "" {
					identities[trimmed] = true
				}
				identityDepth = 0
			}
			if current.Name.Local == "list" && len(listDepths) > 0 && listDepths[len(listDepths)-1] == depth {
				listDepths = listDepths[:len(listDepths)-1]
			}
			depth--
			if depth == 0 {
				rootClosed = true
			}
		}
	}
	if !rootSeen || !rootClosed || depth != 0 {
		return parsedObservation{}, newError("OpenDART returned malformed XML", contextFields)
	}
	return parsedObservation{apiStatus: status, returnedIdentityValues: sortedKeys(identities)}, nil
}

func assertCanonicalObservation(observation Observation) error {
	expectedMedia := "application/" + observation.Request.Format
	if observation.Response.HTTPStatus == http.StatusOK &&
		stringValue(observation.Response.APIStatus) == "000" &&
		stringValue(observation.Response.MediaType) == expectedMedia &&
		len(observation.Response.MissingExpectedIdentityValues) == 0 &&
		len(observation.Response.UnexpectedIdentityValues) == 0 {
		return nil
	}
	return newError("Comma-separated request did not return both guide-example companies", map[string]any{
		"logicalOperationId":            observation.Request.LogicalOperationID,
		"endpoint":                      observation.Request.Endpoint,
		"format":                        observation.Request.Format,
		"httpStatus":                    observation.Response.HTTPStatus,
		"mediaType":                     observation.Response.MediaType,
		"apiStatus":                     observation.Response.APIStatus,
		"missingExpectedIdentityValues": observation.Response.MissingExpectedIdentityValues,
		"unexpectedIdentityValues":      observation.Response.UnexpectedIdentityValues,
	})
}

func distinctBaselineIdentityValues(testCase probeCase, observations []Observation) ([]string, error) {
	identities := make(map[string]bool)
	for index, baseline := range observations {
		if baseline.Response.HTTPStatus != http.StatusOK ||
			stringValue(baseline.Response.APIStatus) != "000" ||
			stringValue(baseline.Response.MediaType) != "application/json" ||
			len(baseline.Response.ReturnedIdentityValues) != 1 {
			return nil, newError("Single-company baseline did not expose one response identity", map[string]any{
				"logicalOperationId":    testCase.LogicalOperationID,
				"endpoint":              testCase.Endpoint,
				"corpCode":              testCase.CorpCodes[index],
				"httpStatus":            baseline.Response.HTTPStatus,
				"mediaType":             baseline.Response.MediaType,
				"apiStatus":             baseline.Response.APIStatus,
				"responseIdentityCount": len(baseline.Response.ReturnedIdentityValues),
			})
		}
		identities[baseline.Response.ReturnedIdentityValues[0]] = true
	}
	distinct := sortedKeys(identities)
	if len(distinct) != len(testCase.CorpCodes) {
		return nil, newError("Single-company baselines did not produce distinct identities", map[string]any{
			"logicalOperationId":    testCase.LogicalOperationID,
			"endpoint":              testCase.Endpoint,
			"requestedCompanyCount": len(testCase.CorpCodes),
			"responseIdentityCount": len(distinct),
		})
	}
	return distinct, nil
}

func repeatedKeyConclusion(observations []Observation) string {
	accepted := 0
	for _, observation := range observations {
		if observation.Response.HTTPStatus == http.StatusOK &&
			stringValue(observation.Response.APIStatus) == "000" &&
			len(observation.Response.MissingExpectedIdentityValues) == 0 &&
			len(observation.Response.UnexpectedIdentityValues) == 0 {
			accepted++
		}
	}
	if accepted == len(observations) {
		return "accepted-in-all-controls"
	}
	if accepted == 0 {
		return "not-accepted-in-controls"
	}
	return "inconsistent-across-controls"
}

func readBoundedBody(body io.ReadCloser) ([]byte, error) {
	limited := io.LimitReader(body, maxResponseBody+1)
	value, readErr := io.ReadAll(limited)
	closeErr := body.Close()
	if readErr != nil {
		return nil, errors.New("OpenDART response body could not be read")
	}
	if closeErr != nil {
		return nil, errors.New("OpenDART response body could not be closed")
	}
	if len(value) > maxResponseBody {
		return nil, errors.New("OpenDART response body exceeded the size limit")
	}
	return value, nil
}

func mediaType(header string) *string {
	if header == "" {
		return nil
	}
	value, _, err := mime.ParseMediaType(header)
	if err != nil || strings.TrimSpace(value) == "" {
		return nil
	}
	value = strings.ToLower(strings.TrimSpace(value))
	return &value
}

func identityDifferences(expected, actual []string) ([]string, []string) {
	if expected == nil {
		return []string{}, []string{}
	}
	expectedSet := make(map[string]bool, len(expected))
	actualSet := make(map[string]bool, len(actual))
	for _, value := range expected {
		expectedSet[value] = true
	}
	for _, value := range actual {
		actualSet[value] = true
	}
	missing := make([]string, 0)
	unexpected := make([]string, 0)
	for _, value := range expected {
		if !actualSet[value] {
			missing = append(missing, value)
		}
	}
	for _, value := range actual {
		if !expectedSet[value] {
			unexpected = append(unexpected, value)
		}
	}
	return missing, unexpected
}

func requestContext(testCase probeCase, format string, encoding serialization) map[string]any {
	return map[string]any{
		"logicalOperationId":    testCase.LogicalOperationID,
		"endpoint":              testCase.Endpoint,
		"format":                format,
		"serialization":         string(encoding),
		"responseIdentityField": testCase.ResponseIdentityField,
	}
}

func newError(message string, contextFields map[string]any) *Error {
	if contextFields == nil {
		contextFields = map[string]any{}
	}
	return &Error{Message: message, Context: contextFields}
}

func checkedAtInSeoul(now time.Time) (string, error) {
	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return "", fmt.Errorf("load timezone: %w", err)
	}
	return now.In(location).Format(time.DateOnly), nil
}

func waitFor(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneMap(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func reportContainsCredential(report Report, key string) (bool, error) {
	encoded, err := json.Marshal(report)
	if err != nil {
		return false, err
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		return false, err
	}
	var contains func(any) bool
	contains = func(current any) bool {
		switch typed := current.(type) {
		case string:
			return strings.Contains(typed, key)
		case []any:
			for _, child := range typed {
				if contains(child) {
					return true
				}
			}
		case map[string]any:
			for _, child := range typed {
				if contains(child) {
					return true
				}
			}
		}
		return false
	}
	return contains(value), nil
}
