package guide

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fetchFunc func(context.Context, *url.URL) ([]byte, error)

func (function fetchFunc) Fetch(ctx context.Context, sourceURL *url.URL) ([]byte, error) {
	return function(ctx, sourceURL)
}

func TestTrustedGuideURL(t *testing.T) {
	t.Parallel()
	accepted := []string{
		"/guide/main.do?apiGrpCd=DS001",
		"https://opendart.fss.or.kr/guide/detail.do?apiGrpCd=DS001&apiId=2019001",
	}
	for _, value := range accepted {
		if _, err := trustedGuideURL(value, ""); err != nil {
			t.Errorf("trustedGuideURL(%q): %v", value, err)
		}
	}
	rejected := []string{
		"http://opendart.fss.or.kr/guide/main.do",
		"https://OPENDART.FSS.OR.KR/guide/main.do",
		"https://opendart.fss.or.kr:443/guide/main.do",
		"https://user@opendart.fss.or.kr/guide/main.do",
		"https://opendart.fss.or.kr/guide/main.do#fragment",
		"https://opendart.fss.or.kr/api/list.json",
		"https://evil.example/guide/main.do",
		"//evil.example/guide/detail.do",
	}
	for _, value := range rejected {
		if _, err := trustedGuideURL(value, ""); err == nil {
			t.Errorf("trustedGuideURL(%q) unexpectedly succeeded", value)
		}
	}
	if _, err := trustedGuideURL("/guide/main.do", "/guide/detail.do"); err == nil {
		t.Fatal("expected exact-path rejection")
	}
}

func TestTrustedGuideURLRedactsRejectedCredentials(t *testing.T) {
	t.Parallel()
	_, err := trustedGuideURL("https://user:secret@evil.example/guide/main.do?token=private", "")
	var source *SourceError
	if !errors.As(err, &source) {
		t.Fatalf("error = %v", err)
	}
	diagnostic := fmt.Sprintf("%v", source.Context)
	for _, secret := range []string{"user", "secret", "private", "token"} {
		if strings.Contains(diagnostic, secret) {
			t.Fatalf("diagnostic %q contains %q", diagnostic, secret)
		}
	}
	if source.Context["host"] != "evil.example" || source.Context["path"] != "/guide/main.do" {
		t.Fatalf("context = %#v", source.Context)
	}
}

func TestEndpointIdentityRequiresSingleMatchingParameters(t *testing.T) {
	t.Parallel()
	sourceURL, group, apiID, err := endpointIdentity("/guide/detail.do?apiGrpCd=DS001&apiId=2019001&view=full", "DS001")
	if err != nil {
		t.Fatal(err)
	}
	if group != "DS001" || apiID != "2019001" || sourceURL.Path != "/guide/detail.do" {
		t.Fatalf("identity = %s %q %q", sourceURL, group, apiID)
	}
	invalid := []string{
		"/guide/detail.do?apiGrpCd=DS002&apiId=2019001",
		"/guide/detail.do?apiGrpCd=DS001&apiGrpCd=DS001&apiId=2019001",
		"/guide/detail.do?apiGrpCd=DS001&apiId=2019001&apiId=2019002",
		"/guide/detail.do?apiGrpCd=DS001&apiId=abc",
	}
	for _, href := range invalid {
		if _, _, _, err := endpointIdentity(href, "DS001"); err == nil {
			t.Errorf("endpointIdentity(%q) unexpectedly succeeded", href)
		}
	}
}

func TestAcquireGroupNormalizesInventoryFixture(t *testing.T) {
	t.Parallel()
	body := readAcquisitionFixture(t, "group-ds001.html")
	fetcher := fetchFunc(func(_ context.Context, sourceURL *url.URL) ([]byte, error) {
		if sourceURL.String() != "https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS001" {
			t.Fatalf("source URL = %s", sourceURL)
		}
		return body, nil
	})

	endpoints, err := acquireGroup(context.Background(), fetcher, Groups[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 4 {
		t.Fatalf("endpoint count = %d", len(endpoints))
	}
	first := endpoints[0]
	if first.LogicalOperationID != "DS001-2019001" || first.Name != "공시검색" || first.Description != "공시 목록을\n조회합니다." {
		t.Fatalf("first endpoint = %#v", first)
	}
	if first.GroupSourceURL != "https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS001" {
		t.Fatalf("group URL = %q", first.GroupSourceURL)
	}
}

func TestAcquireEndpointNormalizesOfficialGuidePrimitives(t *testing.T) {
	t.Parallel()
	body := readAcquisitionFixture(t, "detail.html")
	summary := EndpointSummary{
		APIGroupCode: "DS001", APIGroupName: "공시정보", APIID: "2019001",
		LogicalOperationID: "DS001-2019001", Name: "공시검색", Description: "공시 목록",
		SourceURL:      "https://opendart.fss.or.kr/guide/detail.do?apiGrpCd=DS001&apiId=2019001",
		GroupSourceURL: "https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS001",
	}
	endpoint, err := acquireEndpoint(context.Background(), fetchFunc(func(_ context.Context, _ *url.URL) ([]byte, error) {
		return body, nil
	}), summary)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.PageHeading != "공시검색 개발가이드" {
		t.Fatalf("page heading = %q", endpoint.PageHeading)
	}
	if want := []BasicInfo{{Method: "GET", RequestURL: "https://opendart.fss.or.kr/api/list.json", Encoding: "UTF-8", OutputFormat: "JSON"}}; !reflect.DeepEqual(endpoint.BasicInfo, want) {
		t.Fatalf("basic info = %#v", endpoint.BasicInfo)
	}
	if got := endpoint.RequestArguments[1].Description; got != "공시대상회사\n고유번호" {
		t.Fatalf("request whitespace = %q", got)
	}
	if len(endpoint.ResponseFields) != 3 {
		t.Fatalf("response fields = %#v", endpoint.ResponseFields)
	}
	root := endpoint.ResponseFields[0]
	if root.SourceIndex != 0 || root.Key != "result" || root.Depth == nil || *root.Depth != 0 || root.SourceKind != "container" || root.SourceIndentClass == nil || *root.SourceIndentClass != "tree mgl5" || root.SourceIconClass == nil || *root.SourceIconClass != "iconArrow" {
		t.Fatalf("root field metadata = %#v", root)
	}
	message := endpoint.ResponseFields[2]
	if message.Depth == nil || *message.Depth != 2 || message.Description != "첫 줄\n둘째 줄" {
		t.Fatalf("nested field = %#v", message)
	}
	wantReference := ReferenceTable{
		Title: "보고서 코드", Headers: []string{"구분", "코드", "설명"},
		Rows:          [][]string{{"정기", "사업보고서", "사업보고서"}, {"정기", "11011", "연간"}},
		Normalization: "rowspan-and-colspan-expanded",
	}
	if !reflect.DeepEqual(endpoint.ReferenceTables, []ReferenceTable{wantReference}) {
		t.Fatalf("reference tables = %#v", endpoint.ReferenceTables)
	}
	if want := []SectionNote{{Section: "기본 안내", Text: "첫 번째 안내입니다.\n\n두 번째 안내입니다."}}; !reflect.DeepEqual(endpoint.SectionNotes, want) {
		t.Fatalf("section notes = %#v", endpoint.SectionNotes)
	}
	if want := []GuideTestArgument{{Key: "corp_code", Value: "00126380,00164779"}, {Key: "empty", Value: ""}}; !reflect.DeepEqual(endpoint.GuideTestRequestArguments, want) {
		t.Fatalf("guide test args = %#v", endpoint.GuideTestRequestArguments)
	}
	if want := []MessageCode{{Code: "000", Description: "정상"}, {Code: "013", Description: "조회된 데이터가 없습니다."}}; !reflect.DeepEqual(endpoint.MessageCodes, want) {
		t.Fatalf("message codes = %#v", endpoint.MessageCodes)
	}
	wantHeaders := SourceTableHeaders{
		BasicInfo:        []string{"메서드", "요청URL", "인코딩", "출력포멧"},
		RequestArguments: []string{"요청키", "명칭", "타입", "필수여부", "값설명"},
		ResponseFields:   []string{"응답키", "명칭", "출력설명"},
	}
	if !reflect.DeepEqual(endpoint.SourceTableHeaders, wantHeaders) {
		t.Fatalf("source headers = %#v", endpoint.SourceTableHeaders)
	}
}

func TestAcquireEndpointRejectsIdentityAndTableDrift(t *testing.T) {
	t.Parallel()
	body := string(readAcquisitionFixture(t, "detail.html"))
	summary := EndpointSummary{
		APIGroupCode: "DS001", APIID: "2019001", LogicalOperationID: "DS001-2019001",
		SourceURL: "https://opendart.fss.or.kr/guide/detail.do?apiGrpCd=DS001&apiId=2019001",
	}
	tests := []struct {
		name    string
		replace string
		with    string
		message string
	}{
		{name: "hidden identity", replace: `name="apiId" value="2019001"`, with: `name="apiId" value="2019002"`, message: "Detail page identity does not match its link"},
		{name: "Korean header", replace: "출력포멧", with: "출력형식", message: "Guide table headers changed"},
		{name: "row width", replace: "<td>JSON</td>", with: "", message: "Guide table row width changed"},
		{name: "message code", replace: "000 정상", with: "정상", message: "Message-code row has no three-digit code"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := strings.Replace(body, test.replace, test.with, 1)
			_, err := acquireEndpoint(context.Background(), fetchFunc(func(_ context.Context, _ *url.URL) ([]byte, error) {
				return []byte(changed), nil
			}), summary)
			if err == nil || err.Error() != test.message {
				t.Fatalf("error = %v, want %q", err, test.message)
			}
		})
	}
}

func TestAcquireUsesCompleteInventoryAndBoundedConcurrency(t *testing.T) {
	detail := string(readAcquisitionFixture(t, "detail.html"))
	fetcher := newInventoryFetcher(detail)
	only := []string{
		"DS003-2019002", "DS001-2019001", "DS006-2019001", "DS002-2019003",
		"DS005-2019001", "DS004-2019002", "DS002-2019001", "DS001-2019001",
	}
	endpoints, err := Acquire(context.Background(), fetcher, only)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"DS001-2019001", "DS002-2019001", "DS002-2019003", "DS003-2019002", "DS004-2019002", "DS005-2019001", "DS006-2019001"}
	got := make([]string, len(endpoints))
	for index, endpoint := range endpoints {
		got[index] = endpoint.LogicalOperationID
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("endpoint order = %#v", got)
	}
	if fetcher.maxActive.Load() > acquireWorkers || fetcher.maxActive.Load() < 2 {
		t.Fatalf("max active fetches = %d", fetcher.maxActive.Load())
	}
	if fetcher.groupFetches.Load() != int32(len(Groups)) {
		t.Fatalf("group fetches = %d", fetcher.groupFetches.Load())
	}
	if fetcher.detailFetches.Load() != int32(len(want)) {
		t.Fatalf("detail fetches = %d", fetcher.detailFetches.Load())
	}
}

func TestAcquireRejectsUnknownOnlyBeforeDetails(t *testing.T) {
	t.Parallel()
	fetcher := newInventoryFetcher(string(readAcquisitionFixture(t, "detail.html")))
	_, err := Acquire(context.Background(), fetcher, []string{"DS999-1"})
	if err == nil || err.Error() != "One or more --only identities were not found" {
		t.Fatalf("error = %v", err)
	}
	if fetcher.detailFetches.Load() != 0 {
		t.Fatalf("detail fetches = %d", fetcher.detailFetches.Load())
	}
	var source *SourceError
	if !errors.As(err, &source) || !reflect.DeepEqual(source.Context["missing"], []string{"DS999-1"}) {
		t.Fatalf("source error = %#v", source)
	}
}

func TestAcquireHonorsCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Acquire(ctx, fetchFunc(func(ctx context.Context, _ *url.URL) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}), []string{"DS001-2019001"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestHTTPFetcherRetriesWithoutCredentials(t *testing.T) {
	t.Parallel()
	statuses := []int{http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusOK}
	var requests int
	fetcher := NewHTTPFetcher()
	fetcher.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Accept") != "text/html,application/xhtml+xml" || request.Header.Get("User-Agent") != "opendart-spec/1.0" {
			t.Errorf("headers = %#v", request.Header)
		}
		if request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" || request.URL.User != nil {
			t.Fatalf("request contains credentials: %s %#v", request.URL, request.Header)
		}
		status := statuses[requests]
		requests++
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("guide")), Request: request}, nil
	})
	fetcher.retryDelay = func(context.Context, time.Duration) error { return nil }
	sourceURL, _ := url.Parse("https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS001")
	body, err := fetcher.Fetch(context.Background(), sourceURL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "guide" || requests != 3 {
		t.Fatalf("body = %q, requests = %d", body, requests)
	}
}

func TestHTTPFetcherDoesNotRetryClientErrorsOrFollowRedirects(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusNotFound, http.StatusFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var requests int
			fetcher := NewHTTPFetcher()
			fetcher.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				requests++
				header := make(http.Header)
				header.Set("Location", "https://evil.example/guide/main.do")
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader("not returned")), Request: request}, nil
			})
			fetcher.retryDelay = func(context.Context, time.Duration) error { return nil }
			sourceURL, _ := url.Parse("https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS001")
			body, err := fetcher.Fetch(context.Background(), sourceURL)
			if err == nil || body != nil || requests != 1 {
				t.Fatalf("body = %q, error = %v, requests = %d", body, err, requests)
			}
		})
	}
}

func TestHTTPFetcherPerAttemptTimeout(t *testing.T) {
	t.Parallel()
	fetcher := NewHTTPFetcher()
	fetcher.attempts = 1
	fetcher.timeout = 5 * time.Millisecond
	fetcher.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	sourceURL, _ := url.Parse("https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS001")
	_, err := fetcher.Fetch(context.Background(), sourceURL)
	if err == nil || !strings.Contains(err.Error(), "after retries") {
		t.Fatalf("error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type inventoryFetcher struct {
	detail        string
	active        atomic.Int32
	maxActive     atomic.Int32
	groupFetches  atomic.Int32
	detailFetches atomic.Int32
}

func newInventoryFetcher(detail string) *inventoryFetcher { return &inventoryFetcher{detail: detail} }

func (fetcher *inventoryFetcher) Fetch(ctx context.Context, sourceURL *url.URL) ([]byte, error) {
	active := fetcher.active.Add(1)
	defer fetcher.active.Add(-1)
	for {
		maximum := fetcher.maxActive.Load()
		if active <= maximum || fetcher.maxActive.CompareAndSwap(maximum, active) {
			break
		}
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Millisecond):
	}
	if sourceURL.Path == "/guide/main.do" {
		fetcher.groupFetches.Add(1)
		groupCode := sourceURL.Query().Get("apiGrpCd")
		var group Group
		for _, candidate := range Groups {
			if candidate.Code == groupCode {
				group = candidate
				break
			}
		}
		var builder strings.Builder
		builder.WriteString("<table><tbody>")
		for index := 1; index <= group.ExpectedCount; index++ {
			fmt.Fprintf(&builder, `<tr><td><a href="/guide/detail.do?apiGrpCd=%s&amp;apiId=%d">상세</a></td><td>이름 %d</td><td>설명 %d</td></tr>`, group.Code, 2019000+index, index, index)
		}
		builder.WriteString("</tbody></table>")
		return []byte(builder.String()), nil
	}
	fetcher.detailFetches.Add(1)
	groupCode := sourceURL.Query().Get("apiGrpCd")
	apiID := sourceURL.Query().Get("apiId")
	body := strings.Replace(fetcher.detail, `name="apiGrpCd" value="DS001"`, `name="apiGrpCd" value="`+groupCode+`"`, 1)
	body = strings.Replace(body, `name="apiId" value="2019001"`, `name="apiId" value="`+apiID+`"`, 1)
	return []byte(body), nil
}

func readAcquisitionFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/acquisition/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
