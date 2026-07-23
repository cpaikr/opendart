package guide

import (
	"reflect"
	"testing"
)

func TestStringSchemaPromotesOnlyCuratedRequestConstraints(t *testing.T) {
	tests := []struct {
		name string
		want map[string]any
	}{
		{name: "corp_code", want: map[string]any{"type": "string", "format": "opendart-corp-code", "minLength": 8, "maxLength": 8}},
		{name: "bgn_de", want: map[string]any{"type": "string", "format": "opendart-date", "minLength": 8, "maxLength": 8}},
		{name: "bsns_year", want: map[string]any{"type": "string", "format": "opendart-year", "minLength": 4, "maxLength": 4}},
		{name: "reprt_code", want: map[string]any{"type": "string", "enum": []string{"11013", "11012", "11014", "11011"}}},
		{name: "page_count", want: map[string]any{"type": "string", "x-opendart-decimal-range": map[string]any{"minimum": 1, "maximum": 100}}},
		{name: "rcept_no", want: map[string]any{"type": "string"}},
		{name: "pblntf_detail_ty", want: map[string]any{"type": "string"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := stringSchema(RequestArgument{Key: test.name, Description: "narrative text is not parsed"})
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("stringSchema() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParameterObjectsApplyElementConstraintsToMultiCompanyArrays(t *testing.T) {
	endpoint := Endpoint{
		EndpointSummary:           EndpointSummary{APIGroupCode: "DS003", APIID: "2019017", LogicalOperationID: "DS003-2019017"},
		RequestArguments:          []RequestArgument{{Key: "corp_code", Required: "Y"}},
		GuideTestRequestArguments: []GuideTestArgument{{Key: "corp_code", Value: "00126380,00164779"}},
		MessageCodes:              []MessageCode{{Code: "021", Description: "조회 가능한 회사는 최대 100건"}},
	}
	parameters, err := parameterObjects(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	parameter := parameters[0].(map[string]any)
	schema := parameter["schema"].(map[string]any)
	items := schema["items"].(map[string]any)
	if items["format"] != "opendart-corp-code" || items["minLength"] != 8 || items["maxLength"] != 8 {
		t.Fatalf("array item constraints = %#v", items)
	}
}
