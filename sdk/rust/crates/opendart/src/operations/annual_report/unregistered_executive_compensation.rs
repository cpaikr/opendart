use super::*;

/// Inputs for logical operation `DS002-2020013`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct UnregisteredExecutiveCompensationInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl UnregisteredExecutiveCompensationInput {
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

    /// Prepares physical operation `get_unrstExctvMendngSttus_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<UnregisteredExecutiveCompensationJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_unrstExctvMendngSttus_json", "DS002-2020013");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json(
                "/api/unrstExctvMendngSttus.json",
                operation,
                &parameters,
            ),
            decode_unregistered_executive_compensation_json_response,
        ))
    }

    /// Prepares physical operation `get_unrstExctvMendngSttus_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<UnregisteredExecutiveCompensationXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_unrstExctvMendngSttus_xml", "DS002-2020013");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/unrstExctvMendngSttus.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_unregistered_executive_compensation_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct UnregisteredExecutiveCompensationEntry<'a> {
    source: &'a SourceValue,
}

impl<'a> UnregisteredExecutiveCompensationEntry<'a> {
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

    /// Returns source field `fyer_salary_totamt` when present.
    #[must_use]
    pub fn annual_salary_total(&self) -> Option<&SourceValue> {
        self.source.get("fyer_salary_totamt")
    }

    /// Returns source field `jan_salary_am` when present.
    #[must_use]
    pub fn average_salary_per_employee(&self) -> Option<&SourceValue> {
        self.source.get("jan_salary_am")
    }

    /// Returns source field `nmpr` when present.
    #[must_use]
    pub fn person_count(&self) -> Option<&SourceValue> {
        self.source.get("nmpr")
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

    /// Returns source field `se` when present.
    #[must_use]
    pub fn category(&self) -> Option<&SourceValue> {
        self.source.get("se")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }
}

/// Opaque response for physical operation `get_unrstExctvMendngSttus_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct UnregisteredExecutiveCompensationJsonResponse {
    source: SourceValue,
}

impl UnregisteredExecutiveCompensationJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = UnregisteredExecutiveCompensationEntry<'_>> + '_ {
        items(&self.source, "list", false).map(UnregisteredExecutiveCompensationEntry::new)
    }
}

fn decode_unregistered_executive_compensation_json_response(
    source: SourceValue,
) -> Result<UnregisteredExecutiveCompensationJsonResponse, ResponseDecodeError> {
    decode_unregistered_executive_compensation_json_response_root(&source, "$".to_owned())?;
    Ok(UnregisteredExecutiveCompensationJsonResponse { source })
}

fn decode_unregistered_executive_compensation_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_unregistered_executive_compensation_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_unregistered_executive_compensation_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_unregistered_executive_compensation_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_unregistered_executive_compensation_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_unrstExctvMendngSttus_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct UnregisteredExecutiveCompensationXmlResponse {
    source: SourceValue,
}

impl UnregisteredExecutiveCompensationXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = UnregisteredExecutiveCompensationEntry<'_>> + '_ {
        items(&self.source, "list", true).map(UnregisteredExecutiveCompensationEntry::new)
    }
}

fn decode_unregistered_executive_compensation_xml_response(
    source: SourceValue,
) -> Result<UnregisteredExecutiveCompensationXmlResponse, ResponseDecodeError> {
    decode_unregistered_executive_compensation_xml_response_root(&source, "$".to_owned())?;
    Ok(UnregisteredExecutiveCompensationXmlResponse { source })
}

fn decode_unregistered_executive_compensation_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_unregistered_executive_compensation_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_unregistered_executive_compensation_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_unregistered_executive_compensation_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_unregistered_executive_compensation_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
