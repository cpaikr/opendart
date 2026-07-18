package guide

import (
	"os"
	"reflect"
	"testing"
)

func TestExtractCompatibilityFixture(t *testing.T) {
	fixture, err := os.Open("testdata/detail.html")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()

	extracted, err := ExtractCompatibilityFixture(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if extracted.Heading != "기업개황 개발가이드" {
		t.Fatalf("heading = %q", extracted.Heading)
	}
	wantBasic := [][]string{
		{"메서드", "요청", "요청"},
		{"메서드", "URL", "출력포멧"},
		{"GET", "https://opendart.fss.or.kr/api/company.json", "JSON"},
	}
	if !reflect.DeepEqual(extracted.Tables["기본정보"], wantBasic) {
		t.Fatalf("basic table = %#v", extracted.Tables["기본정보"])
	}
	if got := extracted.Tables["요청인자"][1][1]; got != "공시대상회사\n고유번호" {
		t.Fatalf("normalized cell = %q", got)
	}
	if extracted.Hidden["corp_code"] != "00126380" {
		t.Fatalf("hidden values = %#v", extracted.Hidden)
	}
}
