package liveconformance

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"
	"sort"
	"strings"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

// Preflight derives an executable plan without reading a credential or using a
// network. Exactly one case must cover every physical operation.
func Preflight(spec specification, cases []Case, assertions map[AssertionID]Assertion) (*Plan, error) {
	catalog, err := spec.Operations()
	if err != nil {
		return nil, preflightError("operation-enumeration", "")
	}
	if !slices.Equal(catalog.Servers, []string{TrustedServer}) {
		return nil, preflightError("untrusted-server", "")
	}
	if !slices.Equal(catalog.SecuritySchemes, []openapispec.SecurityScheme{{
		Name: "crtfcKey", Type: "apiKey", Location: "query", ParameterName: "crtfc_key",
	}}) {
		return nil, preflightError("invalid-credential-scheme", "")
	}
	if len(catalog.Operations) == 0 {
		return nil, preflightError("empty-operation-catalog", "")
	}
	if len(cases) > AbsoluteRequestLimit {
		return nil, preflightError("request-budget-exceeded", "")
	}

	operations := make(map[string]openapispec.Operation, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		if operation.Method != http.MethodGet || !validOperationPath(operation.Path) || !safeIdentifier.MatchString(operation.OperationID) || !safeIdentifier.MatchString(operation.LogicalOperationID) || !allowedRepresentation(operation.PrimaryRepresentation) {
			return nil, preflightError("invalid-operation-identity", operation.Identity())
		}
		if !slices.Equal(operation.Servers, []string{TrustedServer}) {
			return nil, preflightError("untrusted-operation-server", operation.Identity())
		}
		if !slices.Equal(operation.SecurityRequirements, []string{"crtfcKey"}) {
			return nil, preflightError("invalid-operation-security", operation.Identity())
		}
		if _, exists := operations[operation.Identity()]; exists {
			return nil, preflightError("duplicate-operation", operation.Identity())
		}
		operations[operation.Identity()] = operation
	}

	prepared := make([]preparedCase, 0, len(cases))
	covered := make(map[string]bool, len(cases))
	caseIDs := make(map[string]bool, len(cases))
	for _, testCase := range cases {
		identity := testCase.operationIdentity()
		operation, exists := operations[identity]
		if !exists {
			return nil, preflightError("unknown-case-operation", identity)
		}
		if !safeIdentifier.MatchString(testCase.ID) || caseIDs[testCase.ID] {
			return nil, preflightError("invalid-case-id", identity)
		}
		caseIDs[testCase.ID] = true
		if covered[identity] {
			return nil, preflightError("duplicate-case-coverage", identity)
		}
		covered[identity] = true
		assertion, exists := assertions[testCase.Assertion]
		if !safeIdentifier.MatchString(string(testCase.Assertion)) || !exists || assertion.Check == nil || !slices.Contains(assertion.Representations, testCase.Representation) {
			return nil, preflightError("invalid-assertion", identity)
		}
		query, err := serializeParameters(operation, testCase.Parameters)
		if err != nil {
			return nil, preflightError("invalid-case-parameters", identity)
		}
		if err := spec.ValidateRequest(testCase.Method, testCase.Path, query); err != nil {
			return nil, preflightError("openapi-request-validation", identity)
		}
		prepared = append(prepared, preparedCase{definition: cloneCase(testCase), operation: operation, query: query, assertion: assertion})
	}
	if len(covered) != len(operations) {
		return nil, preflightError("incomplete-operation-coverage", firstMissingOperation(operations, covered))
	}
	sort.Slice(prepared, func(i, j int) bool {
		return prepared[i].operation.Identity() < prepared[j].operation.Identity()
	})
	return &Plan{specification: spec, cases: prepared, requestBudget: len(prepared)}, nil
}

func validOperationPath(value string) bool {
	return safePath.MatchString(value) && !strings.Contains(value, "//") && path.Clean(value) == value
}

func serializeParameters(operation openapispec.Operation, values map[string][]string) (url.Values, error) {
	parameters := make(map[string]openapispec.Parameter, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		if parameter.Location != "query" {
			return nil, fmt.Errorf("parameter %q uses unsupported location %q", parameter.Name, parameter.Location)
		}
		parameters[parameter.Name] = parameter
	}
	for name := range values {
		if _, exists := parameters[name]; !exists {
			return nil, fmt.Errorf("parameter %q is not declared", name)
		}
	}
	query := make(url.Values)
	for name, parameter := range parameters {
		caseValues, exists := values[name]
		if !exists {
			if parameter.Required {
				return nil, fmt.Errorf("required parameter %q is missing", name)
			}
			continue
		}
		if len(caseValues) == 0 {
			return nil, fmt.Errorf("parameter %q has no values", name)
		}
		for _, value := range caseValues {
			if value == "" {
				return nil, fmt.Errorf("parameter %q has an empty value", name)
			}
		}
		isArray := slices.Contains(parameter.SchemaTypes, "array")
		if !isArray {
			if len(caseValues) != 1 {
				return nil, fmt.Errorf("scalar parameter %q has multiple values", name)
			}
			query.Set(name, caseValues[0])
			continue
		}
		if parameter.Style != "" && parameter.Style != "form" {
			return nil, fmt.Errorf("array parameter %q uses unsupported style %q", name, parameter.Style)
		}
		if parameter.Explode {
			for _, value := range caseValues {
				query.Add(name, value)
			}
		} else {
			query.Set(name, strings.Join(caseValues, ","))
		}
	}
	return query, nil
}

func firstMissingOperation(operations map[string]openapispec.Operation, covered map[string]bool) string {
	missing := make([]string, 0)
	for identity := range operations {
		if !covered[identity] {
			missing = append(missing, identity)
		}
	}
	sort.Strings(missing)
	if len(missing) == 0 {
		return ""
	}
	return missing[0]
}

func cloneCase(source Case) Case {
	clone := source
	clone.Parameters = make(map[string][]string, len(source.Parameters))
	for name, values := range source.Parameters {
		clone.Parameters[name] = append([]string(nil), values...)
	}
	return clone
}

func preflightError(code, operation string) *Error {
	return &Error{Failure: Failure{Code: code, Stage: "preflight", Operation: operation}}
}
