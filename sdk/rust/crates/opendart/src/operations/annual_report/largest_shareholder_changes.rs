use super::*;

/// Inputs for logical operation `DS002-2019008`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct LargestShareholderChangesInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl LargestShareholderChangesInput {
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

    /// Prepares physical operation `get_hyslrChgSttus_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<LargestShareholderChangesJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_hyslrChgSttus_json", "DS002-2019008");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/hyslrChgSttus.json", operation, &parameters),
            decode_largest_shareholder_changes_json_response,
        ))
    }

    /// Prepares physical operation `get_hyslrChgSttus_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<LargestShareholderChangesXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_hyslrChgSttus_xml", "DS002-2019008");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/hyslrChgSttus.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_largest_shareholder_changes_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct LargestShareholderChange<'a> {
    source: &'a SourceValue,
}

impl<'a> LargestShareholderChange<'a> {
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

    /// Returns source field `change_cause` when present.
    #[must_use]
    pub fn change_reason(&self) -> Option<&SourceValue> {
        self.source.get("change_cause")
    }

    /// Returns source field `change_on` when present.
    #[must_use]
    pub fn change_date(&self) -> Option<&SourceValue> {
        self.source.get("change_on")
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

    /// Returns source field `mxmm_shrholdr_nm` when present.
    #[must_use]
    pub fn largest_shareholder_name(&self) -> Option<&SourceValue> {
        self.source.get("mxmm_shrholdr_nm")
    }

    /// Returns source field `posesn_stock_co` when present.
    #[must_use]
    pub fn owned_share_count(&self) -> Option<&SourceValue> {
        self.source.get("posesn_stock_co")
    }

    /// Returns source field `qota_rt` when present.
    #[must_use]
    pub fn ownership_percentage(&self) -> Option<&SourceValue> {
        self.source.get("qota_rt")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rm` when present.
    #[must_use]
    pub fn remarks(&self) -> Option<&SourceValue> {
        self.source.get("rm")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }
}

/// Opaque response for physical operation `get_hyslrChgSttus_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct LargestShareholderChangesJsonResponse {
    source: SourceValue,
}

impl LargestShareholderChangesJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = LargestShareholderChange<'_>> + '_ {
        items(&self.source, "list", false).map(LargestShareholderChange::new)
    }
}

fn decode_largest_shareholder_changes_json_response(
    source: SourceValue,
) -> Result<LargestShareholderChangesJsonResponse, ResponseDecodeError> {
    decode_largest_shareholder_changes_json_response_root(&source, "$".to_owned())?;
    Ok(LargestShareholderChangesJsonResponse { source })
}

fn decode_largest_shareholder_changes_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_largest_shareholder_changes_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_largest_shareholder_changes_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_largest_shareholder_changes_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_largest_shareholder_changes_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_hyslrChgSttus_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct LargestShareholderChangesXmlResponse {
    source: SourceValue,
}

impl LargestShareholderChangesXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = LargestShareholderChange<'_>> + '_ {
        items(&self.source, "list", true).map(LargestShareholderChange::new)
    }
}

fn decode_largest_shareholder_changes_xml_response(
    source: SourceValue,
) -> Result<LargestShareholderChangesXmlResponse, ResponseDecodeError> {
    decode_largest_shareholder_changes_xml_response_root(&source, "$".to_owned())?;
    Ok(LargestShareholderChangesXmlResponse { source })
}

fn decode_largest_shareholder_changes_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_largest_shareholder_changes_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_largest_shareholder_changes_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_largest_shareholder_changes_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_largest_shareholder_changes_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
