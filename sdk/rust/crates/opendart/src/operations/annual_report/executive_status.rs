use super::*;

/// Inputs for logical operation `DS002-2019010`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ExecutiveStatusInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl ExecutiveStatusInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(company_code: String, business_year: String, report_code: String) -> Self {
        Self {
            company_code,
            business_year,
            report_code,
        }
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        let value = CompanyCode::new(operation, "corp_code", &self.company_code)?;
        query.required("corp_code", value.as_str())?;
        let value = BusinessYear::new(operation, "bsns_year", &self.business_year)?;
        query.required("bsns_year", value.as_str())?;
        let value = ReportCode::new(operation, "reprt_code", &self.report_code)?;
        query.required("reprt_code", value.as_str())?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_exctvSttus_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<ExecutiveStatusJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_exctvSttus_json", "DS002-2019010");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/exctvSttus.json", operation, &parameters),
            decode_executive_status_json_response,
        ))
    }

    /// Prepares physical operation `get_exctvSttus_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(&self) -> Result<PreparedRequest<ExecutiveStatusXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_exctvSttus_xml", "DS002-2019010");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/exctvSttus.xml", operation, &parameters, "result"),
            decode_executive_status_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Executive<'a> {
    source: &'a SourceValue,
}

impl<'a> Executive<'a> {
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

    /// Returns source field `birth_ym` when present.
    #[must_use]
    pub fn birth_year_month(&self) -> Option<&SourceValue> {
        self.source.get("birth_ym")
    }

    /// Returns source field `chrg_job` when present.
    #[must_use]
    pub fn responsibilities(&self) -> Option<&SourceValue> {
        self.source.get("chrg_job")
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

    /// Returns source field `fte_at` when present.
    #[must_use]
    pub fn full_time_status(&self) -> Option<&SourceValue> {
        self.source.get("fte_at")
    }

    /// Returns source field `hffc_pd` when present.
    #[must_use]
    pub fn service_period(&self) -> Option<&SourceValue> {
        self.source.get("hffc_pd")
    }

    /// Returns source field `main_career` when present.
    #[must_use]
    pub fn career_summary(&self) -> Option<&SourceValue> {
        self.source.get("main_career")
    }

    /// Returns source field `mxmm_shrholdr_relate` when present.
    #[must_use]
    pub fn relationship_to_largest_shareholder(&self) -> Option<&SourceValue> {
        self.source.get("mxmm_shrholdr_relate")
    }

    /// Returns source field `nm` when present.
    #[must_use]
    pub fn name(&self) -> Option<&SourceValue> {
        self.source.get("nm")
    }

    /// Returns source field `ofcps` when present.
    #[must_use]
    pub fn position(&self) -> Option<&SourceValue> {
        self.source.get("ofcps")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rgist_exctv_at` when present.
    #[must_use]
    pub fn registered_executive_status(&self) -> Option<&SourceValue> {
        self.source.get("rgist_exctv_at")
    }

    /// Returns source field `sexdstn` when present.
    #[must_use]
    pub fn gender(&self) -> Option<&SourceValue> {
        self.source.get("sexdstn")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }

    /// Returns source field `tenure_end_on` when present.
    #[must_use]
    pub fn term_end_date(&self) -> Option<&SourceValue> {
        self.source.get("tenure_end_on")
    }
}

/// Opaque response for physical operation `get_exctvSttus_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct ExecutiveStatusJsonResponse {
    source: SourceValue,
}

impl ExecutiveStatusJsonResponse {
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

    /// Iterates the reviewed `$.list[]` item views.
    pub fn items(&self) -> impl Iterator<Item = Executive<'_>> + '_ {
        items(&self.source, "list", false).map(Executive::new)
    }
}

fn decode_executive_status_json_response(
    source: SourceValue,
) -> Result<ExecutiveStatusJsonResponse, ResponseDecodeError> {
    decode_executive_status_json_response_root(&source, "$".to_owned())?;
    Ok(ExecutiveStatusJsonResponse { source })
}

fn decode_executive_status_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_executive_status_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_executive_status_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_executive_status_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_executive_status_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_exctvSttus_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct ExecutiveStatusXmlResponse {
    source: SourceValue,
}

impl ExecutiveStatusXmlResponse {
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

    /// Iterates the reviewed `$.list[]` item views.
    pub fn items(&self) -> impl Iterator<Item = Executive<'_>> + '_ {
        items(&self.source, "list", true).map(Executive::new)
    }
}

fn decode_executive_status_xml_response(
    source: SourceValue,
) -> Result<ExecutiveStatusXmlResponse, ResponseDecodeError> {
    decode_executive_status_xml_response_root(&source, "$".to_owned())?;
    Ok(ExecutiveStatusXmlResponse { source })
}

fn decode_executive_status_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_executive_status_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_executive_status_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_executive_status_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_executive_status_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
