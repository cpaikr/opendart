package guide

// requestStringConstraints is the closed set of curated request rules that the
// repository promotes into the canonical OpenAPI contract. These rules are
// intentionally keyed by wire name instead of inferred from narrative text or
// the occasionally contradictory documented type column.
type requestStringConstraints struct {
	format         string
	allowedValues  []string
	minimumLength  int
	maximumLength  int
	decimalMinimum int
	decimalMaximum int
}

func constraintsForRequestArgument(argument RequestArgument) requestStringConstraints {
	switch argument.Key {
	case "corp_code":
		return requestStringConstraints{format: "opendart-corp-code", minimumLength: 8, maximumLength: 8}
	case "bgn_de", "end_de":
		return requestStringConstraints{format: "opendart-date", minimumLength: 8, maximumLength: 8}
	case "bsns_year":
		return requestStringConstraints{format: "opendart-year", minimumLength: 4, maximumLength: 4}
	case "reprt_code":
		return requestStringConstraints{allowedValues: []string{"11013", "11012", "11014", "11011"}}
	case "idx_cl_code":
		return requestStringConstraints{allowedValues: []string{"M210000", "M220000", "M230000", "M240000"}}
	case "fs_div":
		return requestStringConstraints{allowedValues: []string{"OFS", "CFS"}}
	case "last_reprt_at":
		return requestStringConstraints{allowedValues: []string{"Y", "N"}}
	case "pblntf_ty":
		return requestStringConstraints{allowedValues: []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}}
	case "corp_cls":
		return requestStringConstraints{allowedValues: []string{"Y", "K", "N", "E"}}
	case "sort":
		return requestStringConstraints{allowedValues: []string{"date", "crp", "rpt"}}
	case "sort_mth":
		return requestStringConstraints{allowedValues: []string{"asc", "desc"}}
	case "page_no":
		return requestStringConstraints{decimalMinimum: 1}
	case "page_count":
		return requestStringConstraints{decimalMinimum: 1, decimalMaximum: 100}
	default:
		return requestStringConstraints{}
	}
}

func stringSchema(argument RequestArgument) map[string]any {
	constraints := constraintsForRequestArgument(argument)
	schema := map[string]any{"type": "string"}
	if constraints.format != "" {
		schema["format"] = constraints.format
	}
	if len(constraints.allowedValues) > 0 {
		schema["enum"] = constraints.allowedValues
	}
	if constraints.minimumLength > 0 {
		schema["minLength"] = constraints.minimumLength
	}
	if constraints.maximumLength > 0 {
		schema["maxLength"] = constraints.maximumLength
	}
	if constraints.decimalMinimum > 0 {
		rangeConstraint := map[string]any{"minimum": constraints.decimalMinimum}
		if constraints.decimalMaximum > 0 {
			rangeConstraint["maximum"] = constraints.decimalMaximum
		}
		schema["x-opendart-decimal-range"] = rangeConstraint
	}
	return schema
}
