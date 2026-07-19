package guide

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAcquireDriftAllowsInventoryAdditionsAndDerivesRequestBudget(t *testing.T) {
	t.Parallel()
	base := newInventoryFetcher(string(readAcquisitionFixture(t, "detail.html")))
	fetcher := fetchFunc(func(ctx context.Context, sourceURL *url.URL) ([]byte, error) {
		body, err := base.Fetch(ctx, sourceURL)
		if err != nil || sourceURL.Path != "/guide/main.do" || sourceURL.Query().Get("apiGrpCd") != "DS001" {
			return body, err
		}
		addition := `<tr><td>5</td><td>새 API</td><td>새 상세기능</td><td><a href="/guide/detail.do?apiGrpCd=DS001&amp;apiId=2099999">상세</a></td></tr>`
		changed := strings.Replace(string(body), "</tbody>", addition+"</tbody>", 1)
		if changed == string(body) {
			t.Fatal("inventory fixture mutation did not match")
		}
		return []byte(changed), nil
	})

	result, err := acquireDriftWithFetcher(context.Background(), fetcher, AbsoluteDriftRequestLimit)
	if err != nil {
		t.Fatal(err)
	}
	wantEndpoints := ExpectedFullTotals.LogicalEndpoints + 1
	wantBudget := RequestBudget{Ceiling: len(Groups) + wantEndpoints, Used: len(Groups) + wantEndpoints}
	if len(result.Endpoints) != wantEndpoints || result.RequestBudget != wantBudget {
		t.Fatalf("result = %d endpoints, budget %#v", len(result.Endpoints), result.RequestBudget)
	}
	if base.groupFetches.Load() != int32(len(Groups)) || base.detailFetches.Load() != int32(wantEndpoints) {
		t.Fatalf("fetches = %d group, %d detail", base.groupFetches.Load(), base.detailFetches.Load())
	}
	if result.Endpoints[Groups[0].ExpectedCount].LogicalOperationID != "DS001-2099999" {
		t.Fatalf("added endpoint = %#v", result.Endpoints[Groups[0].ExpectedCount])
	}
}

func TestAcquireDriftRejectsInventoryAboveAbsoluteCeilingBeforeDetails(t *testing.T) {
	t.Parallel()
	base := newInventoryFetcher(string(readAcquisitionFixture(t, "detail.html")))
	absoluteLimit := len(Groups) + ExpectedFullTotals.LogicalEndpoints - 1

	result, err := acquireDriftWithFetcher(context.Background(), base, absoluteLimit)
	if err == nil || err.Error() != "OpenDART guide inventory exceeds the absolute request ceiling" {
		t.Fatalf("error = %v", err)
	}
	if base.detailFetches.Load() != 0 {
		t.Fatalf("detail fetches = %d", base.detailFetches.Load())
	}
	if result.RequestBudget != (RequestBudget{Ceiling: absoluteLimit, Used: len(Groups)}) {
		t.Fatalf("budget = %#v", result.RequestBudget)
	}
	var source *SourceError
	if !errors.As(err, &source) || source.Context["ceiling"] != absoluteLimit || source.Context["required"] != len(Groups)+ExpectedFullTotals.LogicalEndpoints {
		t.Fatalf("source error = %#v", source)
	}
}

func TestCurrentInventoryPolicyAllowsEndpointRemoval(t *testing.T) {
	t.Parallel()
	body := string(readAcquisitionFixture(t, "group-ds001.html"))
	start := strings.Index(body, "<tr>\n            <td>4</td>")
	if start < 0 {
		t.Fatal("fourth inventory row is missing from fixture")
	}
	endOffset := strings.Index(body[start:], "</tr>")
	if endOffset < 0 {
		t.Fatal("fourth inventory row is not closed")
	}
	body = body[:start] + body[start+endOffset+len("</tr>"):]
	fetcher := fetchFunc(func(context.Context, *url.URL) ([]byte, error) { return []byte(body), nil })

	endpoints, err := acquireGroupWithPolicy(context.Background(), fetcher, Groups[0], currentInventory)
	if err != nil || len(endpoints) != Groups[0].ExpectedCount-1 {
		t.Fatalf("current inventory = %d endpoints, error %v", len(endpoints), err)
	}
	if _, err := acquireGroupWithPolicy(context.Background(), fetcher, Groups[0], acceptedInventory); err == nil || err.Error() != "Endpoint group count changed" {
		t.Fatalf("accepted inventory error = %v", err)
	}
}

func TestDriftHTTPFetcherMakesOneAttempt(t *testing.T) {
	t.Parallel()
	requests := 0
	fetcher := newDriftHTTPFetcher()
	transport, ok := fetcher.client.Transport.(*http.Transport)
	if !ok || !transport.DisableKeepAlives {
		t.Fatalf("transport = %#v", fetcher.client.Transport)
	}
	fetcher.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("unavailable")),
			Request:    request,
		}, nil
	})
	sourceURL, _ := url.Parse("https://opendart.fss.or.kr/guide/main.do?apiGrpCd=DS001")

	_, err := fetcher.Fetch(context.Background(), sourceURL)
	if err == nil || err.Error() != "OpenDART guide request failed" || requests != 1 {
		t.Fatalf("error = %v, requests = %d", err, requests)
	}
}
