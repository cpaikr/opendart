package guide

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const (
	guideOrigin       = "https://opendart.fss.or.kr"
	maxGuideHTMLBytes = 16 << 20
	acquireWorkers    = 6
	guideConnections  = 1
)

var (
	errGuideHTMLTooLarge = errors.New("guide HTML exceeds the size limit")
	guideFetchPaths      = map[string]struct{}{
		"/guide/main.do":   {},
		"/guide/detail.do": {},
	}
	messageCodePattern = regexp.MustCompile(`\d{3}`)
	indentPattern      = regexp.MustCompile(`mgl(\d+)`)
	standardCaptions   = map[string]struct{}{
		"기본 정보":       {},
		"요청 인자":       {},
		"응답 결과":       {},
		"OpenAPI 테스트": {},
		"메시지 설명":      {},
	}
)

// SourceError reports a guide contract violation without including response
// bodies or credentials in the diagnostic.
type SourceError struct {
	Message string
	Context map[string]any
	Cause   error
}

func (e *SourceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *SourceError) Unwrap() error { return e.Cause }

func sourceError(message string, context map[string]any, cause error) error {
	return &SourceError{Message: message, Context: context, Cause: cause}
}

// Fetcher retrieves one already-validated official guide URL. Implementations
// must honor cancellation and must not attach credentials.
type Fetcher interface {
	Fetch(context.Context, *url.URL) ([]byte, error)
}

// HTTPFetcher retrieves official guide HTML with the repository's retry,
// redirect, timeout, and content-size policy.
type HTTPFetcher struct {
	client     *http.Client
	attempts   int
	timeout    time.Duration
	retryDelay func(context.Context, time.Duration) error
}

func NewHTTPFetcher() *HTTPFetcher {
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	// The public, credential-free guide endpoint only supports TLS 1.2 static
	// RSA. Keep that legacy exception confined to this exact-host acquisition
	// client; API probes and every other repository request retain modern
	// defaults. The small pool also reuses the expensive legacy connection.
	transport.MaxConnsPerHost = guideConnections
	transport.MaxIdleConnsPerHost = guideConnections
	transport.TLSClientConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{tls.TLS_RSA_WITH_AES_128_GCM_SHA256},
	}
	return &HTTPFetcher{
		client: &http.Client{
			Transport: transport,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		attempts: 3,
		timeout:  30 * time.Second,
		retryDelay: func(ctx context.Context, duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func (f *HTTPFetcher) Fetch(ctx context.Context, sourceURL *url.URL) ([]byte, error) {
	if sourceURL == nil {
		return nil, sourceError("OpenDART guide URL is required", nil, nil)
	}
	trusted, err := trustedGuideURL(sourceURL.String(), "")
	if err != nil {
		return nil, err
	}
	if f == nil || f.client == nil {
		return nil, sourceError("OpenDART guide HTTP client is not configured", nil, nil)
	}
	attempts := f.attempts
	if attempts < 1 {
		attempts = 3
	}
	timeout := f.timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	delay := f.retryDelay
	if delay == nil {
		delay = NewHTTPFetcher().retryDelay
	}

	var lastError error
	for attempt := 1; attempt <= attempts; attempt++ {
		attemptContext, cancel := context.WithTimeout(ctx, timeout)
		request, requestError := http.NewRequestWithContext(attemptContext, http.MethodGet, trusted.String(), nil)
		if requestError != nil {
			cancel()
			return nil, sourceError("OpenDART guide request could not be created", map[string]any{"url": trusted.String()}, requestError)
		}
		request.Header.Set("Accept", "text/html,application/xhtml+xml")
		request.Header.Set("User-Agent", "opendart-spec/1.0")
		response, requestError := f.client.Do(request)
		if requestError == nil {
			body, readError := readBoundedGuideBody(response.Body)
			closeError := response.Body.Close()
			cancel()
			if errors.Is(readError, errGuideHTMLTooLarge) {
				return nil, sourceError("OpenDART guide response is too large", map[string]any{"url": trusted.String(), "attempt": attempt}, readError)
			}
			if readError != nil {
				lastError = sourceError("OpenDART guide response could not be read", map[string]any{"url": trusted.String(), "attempt": attempt}, readError)
			} else if closeError != nil {
				lastError = sourceError("OpenDART guide response could not be closed", map[string]any{"url": trusted.String(), "attempt": attempt}, closeError)
			} else if response.StatusCode >= 200 && response.StatusCode < 300 {
				return body, nil
			} else {
				retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
				lastError = sourceError("OpenDART guide request failed", map[string]any{
					"url": trusted.String(), "status": response.StatusCode, "attempt": attempt,
				}, nil)
				if !retryable {
					return nil, lastError
				}
			}
		} else {
			cancel()
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastError = requestError
		}

		if attempt < attempts {
			if err := delay(ctx, time.Duration(attempt)*500*time.Millisecond); err != nil {
				return nil, err
			}
		}
	}
	context := map[string]any{"url": trusted.String()}
	if cause := safeTransportCause(lastError); cause != "" {
		context["cause"] = cause
	}
	return nil, sourceError("OpenDART guide request failed after retries", context, lastError)
}

func safeTransportCause(err error) string {
	if err == nil {
		return ""
	}
	var requestError *url.Error
	if errors.As(err, &requestError) {
		err = requestError.Err
	}
	message := err.Error()
	if len(message) > 256 {
		message = message[:256] + "…"
	}
	return message
}

func readBoundedGuideBody(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxGuideHTMLBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxGuideHTMLBytes {
		return nil, fmt.Errorf("%w (%d bytes)", errGuideHTMLTooLarge, maxGuideHTMLBytes)
	}
	return body, nil
}

// Acquire fetches and normalizes the complete official inventory, then fetches
// either every detail page or the requested logical operation IDs. Results are
// returned in official group and inventory order.
func Acquire(ctx context.Context, fetcher Fetcher, only []string) ([]Endpoint, error) {
	if fetcher == nil {
		return nil, sourceError("OpenDART guide fetcher is required", nil, nil)
	}
	inventoryByGroup, err := mapConcurrent(ctx, Groups, acquireWorkers, func(ctx context.Context, group Group) ([]EndpointSummary, error) {
		return acquireGroup(ctx, fetcher, group)
	})
	if err != nil {
		return nil, err
	}
	var inventory []EndpointSummary
	for _, endpoints := range inventoryByGroup {
		inventory = append(inventory, endpoints...)
	}

	selection := make(map[string]struct{}, len(only))
	for _, identity := range only {
		selection[identity] = struct{}{}
	}
	selected := inventory
	if len(selection) > 0 {
		selected = selected[:0]
		for _, endpoint := range inventory {
			if _, ok := selection[endpoint.LogicalOperationID]; ok {
				selected = append(selected, endpoint)
				delete(selection, endpoint.LogicalOperationID)
			}
		}
		if len(selection) > 0 {
			missing := make([]string, 0, len(selection))
			for identity := range selection {
				missing = append(missing, identity)
			}
			slices.Sort(missing)
			return nil, sourceError("One or more --only identities were not found", map[string]any{"missing": missing}, nil)
		}
	}

	endpoints, err := mapConcurrent(ctx, selected, acquireWorkers, func(ctx context.Context, summary EndpointSummary) (Endpoint, error) {
		return acquireEndpoint(ctx, fetcher, summary)
	})
	if err != nil {
		return nil, err
	}
	if err := validateMessageCodes(endpoints); err != nil {
		return nil, err
	}
	if len(only) == 0 {
		if err := validateFullInventory(endpoints); err != nil {
			return nil, err
		}
	}
	return endpoints, nil
}

func acquireGroup(ctx context.Context, fetcher Fetcher, group Group) ([]EndpointSummary, error) {
	mainURL, err := trustedGuideURL(guideOrigin+"/guide/main.do?apiGrpCd="+url.QueryEscape(group.Code), "/guide/main.do")
	if err != nil {
		return nil, err
	}
	body, err := fetcher.Fetch(ctx, mainURL)
	if err != nil {
		return nil, sourceError("OpenDART guide group could not be fetched", map[string]any{"group": group.Code, "sourceUrl": mainURL.String()}, err)
	}
	root, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, sourceError("OpenDART guide group HTML is invalid", map[string]any{"group": group.Code, "sourceUrl": mainURL.String()}, err)
	}
	var endpoints []EndpointSummary
	walk(root, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data != "tr" || !hasAncestorElement(node, "tbody") {
			return
		}
		cells := directChildElements(node, "td")
		if len(cells) < 3 {
			return
		}
		link := firstMatchingDescendant(node, func(candidate *html.Node) bool {
			return candidate.Type == html.ElementNode && candidate.Data == "a" && strings.Contains(attribute(candidate, "href"), "/guide/detail.do")
		})
		if link == nil {
			return
		}
		sourceURL, apiGroupCode, apiID, identityError := endpointIdentity(attribute(link, "href"), group.Code)
		if identityError != nil {
			err = identityError
			return
		}
		endpoints = append(endpoints, EndpointSummary{
			APIGroupCode: apiGroupCode, APIGroupName: group.Name, APIID: apiID,
			LogicalOperationID: apiGroupCode + "-" + apiID,
			Name:               guideNodeText(cells[1]),
			Description:        guideNodeText(cells[2]),
			SourceURL:          sourceURL.String(), GroupSourceURL: mainURL.String(),
		})
	})
	if err != nil {
		return nil, err
	}
	if len(endpoints) != group.ExpectedCount {
		return nil, sourceError("Endpoint group count changed", map[string]any{
			"group": group.Code, "expected": group.ExpectedCount, "actual": len(endpoints), "sourceUrl": mainURL.String(),
		}, nil)
	}
	return endpoints, nil
}

func endpointIdentity(href, expectedGroupCode string) (*url.URL, string, string, error) {
	sourceURL, err := trustedGuideURL(href, "/guide/detail.do")
	if err != nil {
		return nil, "", "", err
	}
	query := sourceURL.Query()
	groupCodes := query["apiGrpCd"]
	apiIDs := query["apiId"]
	if len(groupCodes) != 1 || len(apiIDs) != 1 || groupCodes[0] != expectedGroupCode || !allDigits(apiIDs[0]) {
		return nil, "", "", sourceError("Endpoint link identity does not match its group", map[string]any{
			"group": expectedGroupCode, "sourceUrl": sourceURL.String(),
		}, nil)
	}
	return sourceURL, groupCodes[0], apiIDs[0], nil
}

func trustedGuideURL(value, expectedPath string) (*url.URL, error) {
	base, _ := url.Parse(guideOrigin)
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, sourceError("OpenDART guide URL is invalid", map[string]any{"expectedPath": expectedPath}, err)
	}
	resolved := base.ResolveReference(parsed)
	_, pathAllowed := guideFetchPaths[resolved.Path]
	if expectedPath != "" {
		pathAllowed = resolved.Path == expectedPath
	}
	if resolved.Scheme != "https" || resolved.Host != "opendart.fss.or.kr" || resolved.User != nil || resolved.Fragment != "" || resolved.Opaque != "" || !pathAllowed {
		return nil, sourceError("OpenDART guide URL is outside the trusted guide surface", map[string]any{
			"scheme": resolved.Scheme, "host": resolved.Hostname(), "path": resolved.Path, "expectedPath": expectedPath,
		}, nil)
	}
	query, err := url.ParseQuery(resolved.RawQuery)
	if err != nil {
		return nil, sourceError("OpenDART guide URL query is invalid", map[string]any{
			"path": resolved.Path, "expectedPath": expectedPath,
		}, err)
	}
	requiredKeys := map[string]struct{}{"apiGrpCd": {}}
	if resolved.Path == "/guide/detail.do" {
		requiredKeys["apiId"] = struct{}{}
	}
	if len(query) != len(requiredKeys) {
		return nil, sourceError("OpenDART guide URL query is outside the trusted guide surface", map[string]any{
			"path": resolved.Path, "expectedPath": expectedPath,
		}, nil)
	}
	for key := range requiredKeys {
		if len(query[key]) != 1 {
			return nil, sourceError("OpenDART guide URL query is outside the trusted guide surface", map[string]any{
				"path": resolved.Path, "expectedPath": expectedPath,
			}, nil)
		}
	}
	return resolved, nil
}

func acquireEndpoint(ctx context.Context, fetcher Fetcher, summary EndpointSummary) (Endpoint, error) {
	sourceURL, err := trustedGuideURL(summary.SourceURL, "/guide/detail.do")
	if err != nil {
		return Endpoint{}, err
	}
	body, err := fetcher.Fetch(ctx, sourceURL)
	if err != nil {
		return Endpoint{}, sourceError("OpenDART guide detail could not be fetched", map[string]any{"logicalOperationId": summary.LogicalOperationID, "sourceUrl": summary.SourceURL}, err)
	}
	root, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return Endpoint{}, sourceError("OpenDART guide detail HTML is invalid", map[string]any{"logicalOperationId": summary.LogicalOperationID, "sourceUrl": summary.SourceURL}, err)
	}
	if err := validateHiddenIdentity(root, summary); err != nil {
		return Endpoint{}, err
	}
	tables, err := collectGuideTables(root)
	if err != nil {
		return Endpoint{}, sourceError("Guide table expansion exceeds the cell limit", map[string]any{
			"logicalOperationId": summary.LogicalOperationID, "sourceUrl": summary.SourceURL,
		}, err)
	}
	basic, err := requiredGuideTable(tables, "기본 정보", summary)
	if err != nil {
		return Endpoint{}, err
	}
	requests, err := requiredGuideTable(tables, "요청 인자", summary)
	if err != nil {
		return Endpoint{}, err
	}
	response, err := requiredGuideTable(tables, "응답 결과", summary)
	if err != nil {
		return Endpoint{}, err
	}
	messages, err := requiredGuideTable(tables, "메시지 설명", summary)
	if err != nil {
		return Endpoint{}, err
	}
	for _, table := range []guideTable{response, messages} {
		if table.SourceHasSpans {
			return Endpoint{}, sourceError("Guide table must not use row or column spans", map[string]any{
				"logicalOperationId": summary.LogicalOperationID, "caption": table.Caption, "sourceUrl": summary.SourceURL,
			}, nil)
		}
	}
	if err := validateGuideTable(basic, []string{"메서드", "요청URL", "인코딩", "출력포멧"}, 4, summary); err != nil {
		return Endpoint{}, err
	}
	if err := validateGuideTable(requests, []string{"요청키", "명칭", "타입", "필수여부", "값설명"}, 5, summary); err != nil {
		return Endpoint{}, err
	}
	if err := validateGuideTable(response, []string{"응답키", "명칭", "출력설명"}, 3, summary); err != nil {
		return Endpoint{}, err
	}
	if err := validateGuideTable(messages, nil, 2, summary); err != nil {
		return Endpoint{}, err
	}

	endpoint := Endpoint{EndpointSummary: summary}
	endpoint.PageHeading = textOfFirst(root, func(node *html.Node) bool {
		return node.Data == "p" && hasAncestorClass(node, "DGTopTitle")
	})
	for _, row := range basic.Rows {
		endpoint.BasicInfo = append(endpoint.BasicInfo, BasicInfo{Method: row[0], RequestURL: row[1], Encoding: row[2], OutputFormat: row[3]})
	}
	for _, row := range requests.Rows {
		endpoint.RequestArguments = append(endpoint.RequestArguments, RequestArgument{Key: row[0], Name: row[1], DocumentedType: row[2], Required: row[3], Description: row[4]})
	}
	endpoint.ResponseFields = extractResponseFields(response.Node)
	endpoint.ReferenceTables, err = extractReferenceTables(tables, summary)
	if err != nil {
		return Endpoint{}, err
	}
	endpoint.SectionNotes = extractSectionNotes(root)
	endpoint.SourceTableHeaders = SourceTableHeaders{BasicInfo: basic.Headers, RequestArguments: requests.Headers, ResponseFields: response.Headers}
	endpoint.GuideTestRequestArguments = extractGuideTestArguments(root)
	endpoint.MessageCodes, err = extractMessageCodes(messages.Node, summary)
	if err != nil {
		return Endpoint{}, err
	}
	if len(endpoint.BasicInfo) == 0 {
		return Endpoint{}, sourceError("Endpoint has no documented request URL", map[string]any{"logicalOperationId": summary.LogicalOperationID, "sourceUrl": summary.SourceURL}, nil)
	}
	for _, row := range endpoint.BasicInfo {
		if row.RequestURL == "" {
			return Endpoint{}, sourceError("Endpoint has no documented request URL", map[string]any{"logicalOperationId": summary.LogicalOperationID, "sourceUrl": summary.SourceURL}, nil)
		}
	}
	return endpoint, nil
}

type guideTable struct {
	Node           *html.Node
	Caption        string
	Headers        []string
	Rows           [][]string
	SourceHasSpans bool
}

func collectGuideTables(root *html.Node) ([]guideTable, error) {
	var tables []guideTable
	var expansionErr error
	walk(root, func(node *html.Node) {
		if expansionErr != nil {
			return
		}
		if node.Type != html.ElementNode || node.Data != "table" || !hasAncestorClass(node, "DGCont") {
			return
		}
		caption := firstMatchingDescendant(node, func(candidate *html.Node) bool {
			return candidate.Type == html.ElementNode && candidate.Data == "caption"
		})
		thead := firstDirectOrDescendant(node, "thead")
		var headers []string
		if thead != nil {
			if row := firstDirectOrDescendant(thead, "tr"); row != nil {
				for _, cell := range directChildTableCells(row) {
					headers = append(headers, guideNodeText(cell))
				}
			}
		}
		var bodyRows []*html.Node
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode || child.Data != "tbody" {
				continue
			}
			for row := child.FirstChild; row != nil; row = row.NextSibling {
				if row.Type == html.ElementNode && row.Data == "tr" {
					bodyRows = append(bodyRows, row)
				}
			}
		}
		rows, err := expandRowGroup(bodyRows, guideNodeText)
		if err != nil {
			expansionErr = err
			return
		}
		table := guideTable{Node: node, Headers: headers, Rows: rows}
		if caption != nil {
			table.Caption = guideNodeText(caption)
		}
		walk(node, func(candidate *html.Node) {
			if hasAncestorElement(candidate, "tbody") && (attribute(candidate, "rowspan") != "" || attribute(candidate, "colspan") != "") {
				table.SourceHasSpans = true
			}
		})
		tables = append(tables, table)
	})
	return tables, expansionErr
}

func requiredGuideTable(tables []guideTable, caption string, endpoint EndpointSummary) (guideTable, error) {
	for _, table := range tables {
		if table.Caption == caption {
			return table, nil
		}
	}
	return guideTable{}, sourceError("Required guide table is missing", map[string]any{
		"logicalOperationId": endpoint.LogicalOperationID, "caption": caption, "sourceUrl": endpoint.SourceURL,
	}, nil)
}

func validateGuideTable(table guideTable, expectedHeaders []string, expectedWidth int, endpoint EndpointSummary) error {
	if expectedHeaders != nil && !slices.Equal(table.Headers, expectedHeaders) {
		return sourceError("Guide table headers changed", map[string]any{
			"logicalOperationId": endpoint.LogicalOperationID, "caption": table.Caption,
			"expectedHeaders": expectedHeaders, "actualHeaders": table.Headers, "sourceUrl": endpoint.SourceURL,
		}, nil)
	}
	for rowIndex, row := range table.Rows {
		if len(row) != expectedWidth {
			return sourceError("Guide table row width changed", map[string]any{
				"logicalOperationId": endpoint.LogicalOperationID, "caption": table.Caption, "rowIndex": rowIndex,
				"expectedWidth": expectedWidth, "actualWidth": len(row), "sourceUrl": endpoint.SourceURL,
			}, nil)
		}
	}
	return nil
}

func validateHiddenIdentity(root *html.Node, summary EndpointSummary) error {
	for _, identity := range []struct {
		name string
		want string
	}{{"apiId", summary.APIID}, {"apiGrpCd", summary.APIGroupCode}} {
		var inputs []*html.Node
		walk(root, func(node *html.Node) {
			if node.Type == html.ElementNode && node.Data == "input" && strings.EqualFold(attribute(node, "type"), "hidden") && attribute(node, "name") == identity.name {
				inputs = append(inputs, node)
			}
		})
		if len(inputs) != 1 || attribute(inputs[0], "value") != identity.want {
			return sourceError("Detail page identity does not match its link", map[string]any{
				"logicalOperationId": summary.LogicalOperationID, "identity": identity.name, "sourceUrl": summary.SourceURL,
			}, nil)
		}
	}
	return nil
}

func extractResponseFields(table *html.Node) []ResponseField {
	var fields []ResponseField
	for _, row := range directBodyRows(table) {
		cells := directChildTableCells(row)
		keyCell := tableCell(cells, 0)
		nameCell := tableCell(cells, 1)
		descriptionCell := tableCell(cells, 2)
		indentClass := ""
		if span := firstMatchingDescendant(keyCell, func(node *html.Node) bool {
			return node.Type == html.ElementNode && node.Data == "span" && strings.Contains(attribute(node, "class"), "mgl")
		}); span != nil {
			indentClass = attribute(span, "class")
		}
		var depth *float64
		if match := indentPattern.FindStringSubmatch(indentClass); len(match) == 2 {
			indent, _ := strconv.Atoi(match[1])
			value := float64(indent) / 20
			if indent == 5 {
				value = 0
			}
			depth = &value
		}
		iconClass := ""
		if icon := firstMatchingDescendant(keyCell, func(node *html.Node) bool {
			return node.Type == html.ElementNode && node.Data == "i"
		}); icon != nil {
			iconClass = attribute(icon, "class")
		}
		kind := "field"
		if iconClass == "iconArrow" {
			kind = "container"
		}
		field := ResponseField{SourceIndex: len(fields), Key: guideNodeText(keyCell), Name: guideNodeText(nameCell), Description: guideNodeText(descriptionCell), Depth: depth, SourceKind: kind}
		if indentClass != "" {
			field.SourceIndentClass = &indentClass
		}
		if iconClass != "" {
			field.SourceIconClass = &iconClass
		}
		fields = append(fields, field)
	}
	return fields
}

func extractReferenceTables(tables []guideTable, summary EndpointSummary) ([]ReferenceTable, error) {
	var references []ReferenceTable
	for _, table := range tables {
		if table.Caption == "" {
			continue
		}
		if _, standard := standardCaptions[table.Caption]; standard {
			continue
		}
		if err := validateGuideTable(table, nil, len(table.Headers), summary); err != nil {
			return nil, err
		}
		normalization := "none"
		if table.SourceHasSpans {
			normalization = "rowspan-and-colspan-expanded"
		}
		references = append(references, ReferenceTable{Title: table.Caption, Headers: table.Headers, Rows: table.Rows, Normalization: normalization})
	}
	return references, nil
}

func extractMessageCodes(table *html.Node, summary EndpointSummary) ([]MessageCode, error) {
	var messages []MessageCode
	for _, row := range directBodyRows(table) {
		cells := directChildTableCells(row)
		if len(cells) < 2 {
			continue
		}
		label := guideNodeText(cells[0])
		code := messageCodePattern.FindString(label)
		if code == "" {
			return nil, sourceError("Message-code row has no three-digit code", map[string]any{
				"label": label, "logicalOperationId": summary.LogicalOperationID, "sourceUrl": summary.SourceURL,
			}, nil)
		}
		messages = append(messages, MessageCode{Code: code, Description: guideNodeText(cells[1])})
	}
	return messages, nil
}

func extractGuideTestArguments(root *html.Node) []GuideTestArgument {
	table := firstMatchingDescendant(root, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "table" && attribute(node, "id") == "testTable"
	})
	if table == nil {
		return nil
	}
	var arguments []GuideTestArgument
	walk(table, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data != "input" || !strings.EqualFold(attribute(node, "type"), "hidden") {
			return
		}
		name := attribute(node, "name")
		if name != "" && name != "crtfc_key" {
			arguments = append(arguments, GuideTestArgument{Key: name, Value: attribute(node, "value")})
		}
	})
	return arguments
}

func extractSectionNotes(root *html.Node) []SectionNote {
	var notes []SectionNote
	walk(root, func(node *html.Node) {
		if node.Type != html.ElementNode || !hasClass(node, "DGCont") {
			return
		}
		headingWrap := firstMatchingDescendant(node, func(candidate *html.Node) bool { return hasClass(candidate, "titleWrapToggle") })
		content := firstMatchingDescendant(node, func(candidate *html.Node) bool { return hasClass(candidate, "contWrapToggle") })
		if headingWrap == nil || content == nil {
			return
		}
		heading := textOfFirst(headingWrap, func(candidate *html.Node) bool { return candidate.Data == "h5" })
		text := filteredGuideNodeText(content, map[string]struct{}{
			"table": {}, "form": {}, "script": {}, "style": {}, "button": {}, "input": {}, "select": {}, "textarea": {},
		})
		if heading != "" && text != "" {
			notes = append(notes, SectionNote{Section: heading, Text: text})
		}
	})
	return notes
}

func validateMessageCodes(endpoints []Endpoint) error {
	if len(endpoints) == 0 {
		return nil
	}
	baseline := endpoints[0].MessageCodes
	for _, endpoint := range endpoints {
		if !reflect.DeepEqual(baseline, endpoint.MessageCodes) {
			return sourceError("Endpoint message-code tables differ", map[string]any{
				"logicalOperationId": endpoint.LogicalOperationID, "sourceUrl": endpoint.SourceURL,
			}, nil)
		}
	}
	return nil
}

func validateFullInventory(endpoints []Endpoint) error {
	totals := InventoryTotals{LogicalEndpoints: len(endpoints)}
	for _, endpoint := range endpoints {
		totals.PhysicalPaths += len(endpoint.BasicInfo)
		totals.RequestArguments += len(endpoint.RequestArguments)
		totals.ResponseFields += len(endpoint.ResponseFields)
	}
	if len(endpoints) > 0 {
		totals.MessageCodes = len(endpoints[0].MessageCodes)
	}
	if totals != ExpectedFullTotals {
		return sourceError("Official guide inventory changed", map[string]any{"expected": ExpectedFullTotals, "actual": totals}, nil)
	}
	return nil
}

func mapConcurrent[I, O any](ctx context.Context, items []I, limit int, task func(context.Context, I) (O, error)) ([]O, error) {
	results := make([]O, len(items))
	if len(items) == 0 {
		return results, nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	var wait sync.WaitGroup
	var once sync.Once
	var firstError error
	workers := min(limit, len(items))
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := range jobs {
				result, err := task(ctx, items[index])
				if err != nil {
					once.Do(func() { firstError = err; cancel() })
					continue
				}
				results[index] = result
			}
		}()
	}
send:
	for index := range items {
		select {
		case jobs <- index:
		case <-ctx.Done():
			break send
		}
	}
	close(jobs)
	wait.Wait()
	if firstError != nil {
		return nil, firstError
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func guideNodeText(node *html.Node) string {
	if node == nil {
		return ""
	}
	return filteredGuideNodeText(node, nil)
}

func filteredGuideNodeText(node *html.Node, excluded map[string]struct{}) string {
	var builder strings.Builder
	var visit func(*html.Node)
	visit = func(current *html.Node) {
		if current.Type == html.ElementNode {
			if _, skip := excluded[current.Data]; skip {
				return
			}
			if current.Data == "br" {
				builder.WriteByte('\n')
				return
			}
		}
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if current.Type == html.ElementNode && (current.Data == "p" || current.Data == "li") {
			builder.WriteByte('\n')
		}
	}
	visit(node)
	return normalizeGuideText(builder.String())
}

func normalizeGuideText(value string) string {
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.ReplaceAll(value, "\r", "")
	lines := strings.Split(value, "\n")
	compact := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" || (len(compact) > 0 && compact[len(compact)-1] != "") {
			compact = append(compact, line)
		}
	}
	return strings.TrimSpace(strings.Join(compact, "\n"))
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func directChildElements(node *html.Node, element string) []*html.Node {
	var children []*html.Node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == element {
			children = append(children, child)
		}
	}
	return children
}

func directChildTableCells(row *html.Node) []*html.Node {
	var cells []*html.Node
	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && (child.Data == "th" || child.Data == "td") {
			cells = append(cells, child)
		}
	}
	return cells
}

func directBodyRows(table *html.Node) []*html.Node {
	var rows []*html.Node
	for child := table.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || child.Data != "tbody" {
			continue
		}
		for row := child.FirstChild; row != nil; row = row.NextSibling {
			if row.Type == html.ElementNode && row.Data == "tr" {
				rows = append(rows, row)
			}
		}
	}
	return rows
}

func firstMatchingDescendant(root *html.Node, match func(*html.Node) bool) *html.Node {
	if root == nil {
		return nil
	}
	var found *html.Node
	walk(root, func(node *html.Node) {
		if found == nil && node != root && match(node) {
			found = node
		}
	})
	return found
}

func tableCell(cells []*html.Node, index int) *html.Node {
	if index < 0 || index >= len(cells) {
		return nil
	}
	return cells[index]
}

func firstDirectOrDescendant(root *html.Node, element string) *html.Node {
	if root.Type == html.ElementNode && root.Data == element {
		return root
	}
	return firstMatchingDescendant(root, func(node *html.Node) bool { return node.Type == html.ElementNode && node.Data == element })
}

func textOfFirst(root *html.Node, match func(*html.Node) bool) string {
	node := firstMatchingDescendant(root, func(node *html.Node) bool { return node.Type == html.ElementNode && match(node) })
	if node == nil {
		return ""
	}
	return guideNodeText(node)
}

func hasAncestorElement(node *html.Node, element string) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type == html.ElementNode && parent.Data == element {
			return true
		}
	}
	return false
}

func hasAncestorClass(node *html.Node, class string) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if hasClass(parent, class) {
			return true
		}
	}
	return false
}

func hasClass(node *html.Node, class string) bool {
	return slices.Contains(strings.Fields(attribute(node, "class")), class)
}
