package guide

// Group is one official OpenDART guide inventory group.
type Group struct {
	Code          string
	Name          string
	ExpectedCount int
}

var Groups = []Group{
	{Code: "DS001", Name: "공시정보", ExpectedCount: 4},
	{Code: "DS002", Name: "정기보고서 주요정보", ExpectedCount: 30},
	{Code: "DS003", Name: "정기보고서 재무정보", ExpectedCount: 7},
	{Code: "DS004", Name: "지분공시 종합정보", ExpectedCount: 2},
	{Code: "DS005", Name: "주요사항보고서 주요정보", ExpectedCount: 36},
	{Code: "DS006", Name: "증권신고서 주요정보", ExpectedCount: 6},
}

type EndpointSummary struct {
	APIGroupCode       string
	APIGroupName       string
	APIID              string
	LogicalOperationID string
	Name               string
	Description        string
	SourceURL          string
	GroupSourceURL     string
}

type BasicInfo struct {
	Method       string `yaml:"method"`
	RequestURL   string `yaml:"requestUrl"`
	Encoding     string `yaml:"encoding"`
	OutputFormat string `yaml:"outputFormat"`
}

type RequestArgument struct {
	Key            string `yaml:"key"`
	Name           string `yaml:"name"`
	DocumentedType string `yaml:"documentedType"`
	Required       string `yaml:"required"`
	Description    string `yaml:"description"`
}

type ResponseField struct {
	SourceIndex       int      `yaml:"sourceIndex"`
	Key               string   `yaml:"key"`
	Name              string   `yaml:"name"`
	Description       string   `yaml:"description"`
	Depth             *float64 `yaml:"depth"`
	SourceIndentClass *string  `yaml:"sourceIndentClass"`
	SourceIconClass   *string  `yaml:"sourceIconClass"`
	SourceKind        string   `yaml:"sourceKind"`
}

type ReferenceTable struct {
	Title         string     `yaml:"title"`
	Headers       []string   `yaml:"headers"`
	Rows          [][]string `yaml:"rows"`
	Normalization string     `yaml:"normalization"`
}

type SectionNote struct {
	Section string `yaml:"section"`
	Text    string `yaml:"text"`
}

type SourceTableHeaders struct {
	BasicInfo        []string `yaml:"basicInfo"`
	RequestArguments []string `yaml:"requestArguments"`
	ResponseFields   []string `yaml:"responseFields"`
}

type GuideTestArgument struct {
	Key   string
	Value string
}

type MessageCode struct {
	Code        string
	Description string
}

// Endpoint is the parser-independent guide model consumed by generation.
type Endpoint struct {
	EndpointSummary
	PageHeading               string
	BasicInfo                 []BasicInfo
	RequestArguments          []RequestArgument
	ResponseFields            []ResponseField
	ReferenceTables           []ReferenceTable
	SectionNotes              []SectionNote
	SourceTableHeaders        SourceTableHeaders
	GuideTestRequestArguments []GuideTestArgument
	MessageCodes              []MessageCode
}

type InventoryTotals struct {
	LogicalEndpoints int
	PhysicalPaths    int
	RequestArguments int
	ResponseFields   int
	MessageCodes     int
}

var ExpectedFullTotals = InventoryTotals{
	LogicalEndpoints: 85,
	PhysicalPaths:    167,
	RequestArguments: 337,
	ResponseFields:   2383,
	MessageCodes:     13,
}
