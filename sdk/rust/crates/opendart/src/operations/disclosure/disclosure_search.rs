use super::*;

/// Inputs for logical operation `DS001-2019001`.
#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct DisclosureSearchInput {
    company_code: Option<String>,
    start_date: Option<String>,
    end_date: Option<String>,
    final_reports_only: Option<String>,
    disclosure_type: Option<String>,
    disclosure_detail_type: Option<String>,
    company_class: Option<String>,
    sort_by: Option<String>,
    sort_order: Option<String>,
    page: Option<String>,
    page_size: Option<String>,
}

impl DisclosureSearchInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new() -> Self {
        Self {
            company_code: None,
            start_date: None,
            end_date: None,
            final_reports_only: None,
            disclosure_type: None,
            disclosure_detail_type: None,
            company_class: None,
            sort_by: None,
            sort_order: None,
            page: None,
            page_size: None,
        }
    }

    /// Sets the optional `corp_code` source input.
    #[must_use]
    pub fn with_company_code(mut self, value: String) -> Self {
        self.company_code = Some(value);
        self
    }

    /// Sets the optional `bgn_de` source input.
    #[must_use]
    pub fn with_start_date(mut self, value: String) -> Self {
        self.start_date = Some(value);
        self
    }

    /// Sets the optional `end_de` source input.
    #[must_use]
    pub fn with_end_date(mut self, value: String) -> Self {
        self.end_date = Some(value);
        self
    }

    /// Sets the optional `last_reprt_at` source input.
    #[must_use]
    pub fn with_final_reports_only(mut self, value: String) -> Self {
        self.final_reports_only = Some(value);
        self
    }

    /// Sets the optional `pblntf_ty` source input.
    #[must_use]
    pub fn with_disclosure_type(mut self, value: String) -> Self {
        self.disclosure_type = Some(value);
        self
    }

    /// Sets the optional `pblntf_detail_ty` source input.
    #[must_use]
    pub fn with_disclosure_detail_type(mut self, value: String) -> Self {
        self.disclosure_detail_type = Some(value);
        self
    }

    /// Sets the optional `corp_cls` source input.
    #[must_use]
    pub fn with_company_class(mut self, value: String) -> Self {
        self.company_class = Some(value);
        self
    }

    /// Sets the optional `sort` source input.
    #[must_use]
    pub fn with_sort_by(mut self, value: String) -> Self {
        self.sort_by = Some(value);
        self
    }

    /// Sets the optional `sort_mth` source input.
    #[must_use]
    pub fn with_sort_order(mut self, value: String) -> Self {
        self.sort_order = Some(value);
        self
    }

    /// Sets the optional `page_no` source input.
    #[must_use]
    pub fn with_page(mut self, value: String) -> Self {
        self.page = Some(value);
        self
    }

    /// Sets the optional `page_count` source input.
    #[must_use]
    pub fn with_page_size(mut self, value: String) -> Self {
        self.page_size = Some(value);
        self
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        let value = self
            .company_code
            .as_deref()
            .map(|value| CompanyCode::new(operation, "corp_code", value))
            .transpose()?;
        query.optional("corp_code", value.map(CompanyCode::as_str))?;
        let value = self
            .start_date
            .as_deref()
            .map(|value| CompactDate::new(operation, "bgn_de", value))
            .transpose()?;
        query.optional("bgn_de", value.map(CompactDate::as_str))?;
        let value = self
            .end_date
            .as_deref()
            .map(|value| CompactDate::new(operation, "end_de", value))
            .transpose()?;
        query.optional("end_de", value.map(CompactDate::as_str))?;
        query.optional("last_reprt_at", self.final_reports_only.as_deref())?;
        if let Some(value) = self.final_reports_only.as_deref() {
            crate::validation::require_allowed(operation, "last_reprt_at", value, &["Y", "N"])?;
        }
        query.optional("pblntf_ty", self.disclosure_type.as_deref())?;
        if let Some(value) = self.disclosure_type.as_deref() {
            crate::validation::require_allowed(
                operation,
                "pblntf_ty",
                value,
                &["A", "B", "C", "D", "E", "F", "G", "H", "I", "J"],
            )?;
        }
        query.optional("pblntf_detail_ty", self.disclosure_detail_type.as_deref())?;
        query.optional("corp_cls", self.company_class.as_deref())?;
        if let Some(value) = self.company_class.as_deref() {
            crate::validation::require_allowed(
                operation,
                "corp_cls",
                value,
                &["Y", "K", "N", "E"],
            )?;
        }
        query.optional("sort", self.sort_by.as_deref())?;
        if let Some(value) = self.sort_by.as_deref() {
            crate::validation::require_allowed(operation, "sort", value, &["date", "crp", "rpt"])?;
        }
        query.optional("sort_mth", self.sort_order.as_deref())?;
        if let Some(value) = self.sort_order.as_deref() {
            crate::validation::require_allowed(operation, "sort_mth", value, &["asc", "desc"])?;
        }
        query.optional("page_no", self.page.as_deref())?;
        if let Some(value) = self.page.as_deref() {
            crate::validation::require_decimal_range(operation, "page_no", value, 1, None)?;
        }
        query.optional("page_count", self.page_size.as_deref())?;
        if let Some(value) = self.page_size.as_deref() {
            crate::validation::require_decimal_range(operation, "page_count", value, 1, Some(100))?;
        }
        Ok(query.finish())
    }

    /// Prepares physical operation `get_list_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<DisclosureSearchJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_list_json", "DS001-2019001");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/list.json", operation, &parameters),
            decode_disclosure_search_json_response,
        ))
    }

    /// Prepares physical operation `get_list_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<DisclosureSearchXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_list_xml", "DS001-2019001");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/list.xml", operation, &parameters, "result"),
            decode_disclosure_search_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DisclosureSummary<'a> {
    source: &'a SourceValue,
}

impl<'a> DisclosureSummary<'a> {
    fn new(source: &'a SourceValue) -> Self {
        Self { source }
    }

    /// Returns the complete normalized source object for this view.
    #[must_use]
    pub const fn source(&self) -> &'a SourceValue {
        self.source
    }

    /// Returns a field by its exact source name.
    #[must_use]
    pub fn field(&self, name: &str) -> Option<&'a SourceValue> {
        self.source.get(name)
    }

    /// Returns source field `corp_cls` when present.
    #[must_use]
    pub fn company_class(&self) -> Option<&SourceValue> {
        self.source.get("corp_cls")
    }

    /// Returns source field `corp_code` when present.
    #[must_use]
    pub fn company_code(&self) -> Option<&SourceValue> {
        self.source.get("corp_code")
    }

    /// Returns source field `corp_name` when present.
    #[must_use]
    pub fn company_name(&self) -> Option<&SourceValue> {
        self.source.get("corp_name")
    }

    /// Returns source field `flr_nm` when present.
    #[must_use]
    pub fn filer_name(&self) -> Option<&SourceValue> {
        self.source.get("flr_nm")
    }

    /// Returns source field `rcept_dt` when present.
    #[must_use]
    pub fn receipt_date(&self) -> Option<&SourceValue> {
        self.source.get("rcept_dt")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `report_nm` when present.
    #[must_use]
    pub fn report_name(&self) -> Option<&SourceValue> {
        self.source.get("report_nm")
    }

    /// Returns source field `rm` when present.
    #[must_use]
    pub fn remarks(&self) -> Option<&SourceValue> {
        self.source.get("rm")
    }

    /// Returns source field `stock_code` when present.
    #[must_use]
    pub fn stock_code(&self) -> Option<&SourceValue> {
        self.source.get("stock_code")
    }
}

/// Opaque response for physical operation `get_list_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DisclosureSearchJsonResponse {
    source: SourceValue,
}

impl DisclosureSearchJsonResponse {
    /// Returns the complete normalized source object.
    #[must_use]
    pub const fn source(&self) -> &SourceValue {
        &self.source
    }

    /// Returns the open source status when present and string-shaped.
    #[must_use]
    pub fn status(&self) -> Option<SourceStatus> {
        source_status(&self.source)
    }

    /// Returns the source message when present.
    #[must_use]
    pub fn message(&self) -> Option<&SourceValue> {
        self.source.get("message")
    }

    /// Returns a field by its exact source name.
    #[must_use]
    pub fn field(&self, name: &str) -> Option<&SourceValue> {
        self.source.get(name)
    }

    /// Returns source field `page_count` when present.
    #[must_use]
    pub fn page_size(&self) -> Option<&SourceValue> {
        self.source.get("page_count")
    }

    /// Returns source field `page_no` when present.
    #[must_use]
    pub fn page_number(&self) -> Option<&SourceValue> {
        self.source.get("page_no")
    }

    /// Returns source field `total_count` when present.
    #[must_use]
    pub fn total_count(&self) -> Option<&SourceValue> {
        self.source.get("total_count")
    }

    /// Returns source field `total_page` when present.
    #[must_use]
    pub fn total_pages(&self) -> Option<&SourceValue> {
        self.source.get("total_page")
    }

    /// Iterates the reviewed `$.list[]` item views.
    pub fn items(&self) -> impl Iterator<Item = DisclosureSummary<'_>> + '_ {
        items(&self.source, "list", false).map(DisclosureSummary::new)
    }
}

fn decode_disclosure_search_json_response(
    source: SourceValue,
) -> Result<DisclosureSearchJsonResponse, ResponseDecodeError> {
    decode_disclosure_search_json_response_root(&source, "$".to_owned())?;
    Ok(DisclosureSearchJsonResponse { source })
}

fn decode_disclosure_search_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_disclosure_search_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_disclosure_search_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_disclosure_search_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_disclosure_search_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_list_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DisclosureSearchXmlResponse {
    source: SourceValue,
}

impl DisclosureSearchXmlResponse {
    /// Returns the complete normalized source object.
    #[must_use]
    pub const fn source(&self) -> &SourceValue {
        &self.source
    }

    /// Returns the open source status when present and string-shaped.
    #[must_use]
    pub fn status(&self) -> Option<SourceStatus> {
        source_status(&self.source)
    }

    /// Returns the source message when present.
    #[must_use]
    pub fn message(&self) -> Option<&SourceValue> {
        self.source.get("message")
    }

    /// Returns a field by its exact source name.
    #[must_use]
    pub fn field(&self, name: &str) -> Option<&SourceValue> {
        self.source.get(name)
    }

    /// Returns source field `page_count` when present.
    #[must_use]
    pub fn page_size(&self) -> Option<&SourceValue> {
        self.source.get("page_count")
    }

    /// Returns source field `page_no` when present.
    #[must_use]
    pub fn page_number(&self) -> Option<&SourceValue> {
        self.source.get("page_no")
    }

    /// Returns source field `total_count` when present.
    #[must_use]
    pub fn total_count(&self) -> Option<&SourceValue> {
        self.source.get("total_count")
    }

    /// Returns source field `total_page` when present.
    #[must_use]
    pub fn total_pages(&self) -> Option<&SourceValue> {
        self.source.get("total_page")
    }

    /// Iterates the reviewed `$.list[]` item views.
    pub fn items(&self) -> impl Iterator<Item = DisclosureSummary<'_>> + '_ {
        items(&self.source, "list", true).map(DisclosureSummary::new)
    }
}

fn decode_disclosure_search_xml_response(
    source: SourceValue,
) -> Result<DisclosureSearchXmlResponse, ResponseDecodeError> {
    decode_disclosure_search_xml_response_root(&source, "$".to_owned())?;
    Ok(DisclosureSearchXmlResponse { source })
}

fn decode_disclosure_search_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_disclosure_search_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_disclosure_search_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_disclosure_search_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_disclosure_search_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
